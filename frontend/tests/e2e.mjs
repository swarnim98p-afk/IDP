import { chromium } from '@playwright/test';
import assert from 'node:assert/strict';
import { mkdir } from 'node:fs/promises';
import path from 'node:path';

const baseURL = process.env.PORTAL_TEST_URL || 'http://127.0.0.1:8080';
const artifactDir = path.resolve('../.run/browser');
await mkdir(artifactDir, { recursive: true });
const browser = await chromium.launch({ headless: true });
try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 }, acceptDownloads: true });
  const errors = []; page.on('pageerror', error => errors.push(error.message));
  await page.goto(baseURL);
  await page.getByText('Services connected', { exact: true }).waitFor();
  assert.equal(await page.getByRole('radio').count(), 2);
  for (const [templateID, suffix] of [['go-gin-api','api'], ['react-frontend','ui']]) {
    const name = `browser-${suffix}-${Date.now()}`;
    await page.locator(`input[value="${templateID}"]`).check();
    await page.getByLabel('Project name', { exact: true }).fill(name);
    const downloadPromise = page.waitForEvent('download');
    await page.getByRole('button', { name: 'Create project', exact: true }).click();
    const download = await downloadPromise;
    assert.match(download.suggestedFilename(), /^project-[a-f0-9]{32}\.zip$/);
    await download.saveAs(path.join(artifactDir, `${suffix}.zip`));
    await page.getByRole('button', { name: `Download ${name}`, exact: true }).waitFor();
    await page.reload(); await page.getByText('Services connected', { exact: true }).waitFor();
    await page.getByRole('button', { name: `Download ${name}`, exact: true }).waitFor();
  }
  await page.getByRole('button', { name: 'Templates', exact: true }).click();
  assert.equal(await page.getByRole('button', { name: 'Use this template', exact: true }).count(), 2);
  await page.getByRole('button', { name: 'Use this template', exact: true }).last().click();
  assert.equal(await page.locator('input[value="react-frontend"]').isChecked(), true);
  await page.getByRole('button', { name: 'Projects', exact: false }).first().click();
  await page.getByRole('textbox', { name: 'Search projects', exact: true }).fill('this-project-does-not-exist');
  await page.getByText('No matching projects', { exact: true }).waitFor();
  await page.getByRole('textbox', { name: 'Search projects', exact: true }).fill('');
  await page.getByRole('button', { name: 'Overview', exact: true }).click();
  await page.screenshot({ path: path.join(artifactDir, 'desktop.png'), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true, 'mobile layout overflows');
  await page.screenshot({ path: path.join(artifactDir, 'mobile.png'), fullPage: true });
  assert.deepEqual(errors, [], 'browser JavaScript errors');
  console.log('PASS: Go and React creation/download, reload persistence, navigation, search, desktop and mobile layouts.');
} finally { await browser.close(); }
