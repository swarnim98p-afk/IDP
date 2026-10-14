import React, { useEffect, useRef, useState } from 'react';

function Icon({ name, size = 20 }) {
  const paths = { grid: 'M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z', box: 'M3 7l9-4 9 4-9 5z M3 7v10l9 4 9-4V7 M12 12v9', layers: 'M3 7l9-4 9 4-9 4z M3 12l9 4 9-4 M3 17l9 4 9-4', arrow: 'M5 12h14 M13 6l6 6-6 6', download: 'M12 3v12 M7 10l5 5 5-5 M4 16v5h16v-5', check: 'M5 12l4 4L19 6', search: 'M11 3a8 8 0 1 0 0 16 8 8 0 0 0 0-16 M17 17l4 4', refresh: 'M20 7a9 9 0 1 0 1 9 M20 3v5h-5', code: 'M8 5l-7 7 7 7 M16 5l7 7-7 7 M14 3l-4 18', plus: 'M12 4v16 M4 12h16', globe: 'M3 12h18 M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 3c-5 5-5 13 0 18 M12 3c5 5 5 13 0 18', clock: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18 M12 7v5l3 2', close: 'M6 6l12 12 M18 6 6 18' };
  return <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d={paths[name] || paths.box} /></svg>;
}

async function requestJSON(path, options = {}) {
  const response = await fetch(path, options);
  let body;
  try { body = await response.json(); } catch { throw new Error('The service returned an unexpected response. Please try again.'); }
  if (!response.ok) throw new Error(body.error || 'The request could not be completed.');
  return body;
}

function dateLabel(value) { return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(new Date(value)); }

export default function App() {
  const [view, setView] = useState('overview');
  const [templates, setTemplates] = useState([]);
  const [projects, setProjects] = useState([]);
  const [cursor, setCursor] = useState('');
  const [selected, setSelected] = useState('go-gin-api');
  const [name, setName] = useState('');
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [downloadID, setDownloadID] = useState('');
  const [autoDownload, setAutoDownload] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const formRef = useRef(null);
  const validName = [...name.trim()].length >= 3 && [...name.trim()].length <= 50;

  async function load() {
    setLoading(true); setError('');
    try {
      const [templateData, projectData, serviceStatus] = await Promise.all([requestJSON('/api/templates'), requestJSON('/api/projects?limit=100'), requestJSON('/api/status')]);
      setTemplates(templateData.templates); setProjects(projectData.projects); setCursor(projectData.nextCursor || ''); setStatus(serviceStatus);
    } catch (err) { setError(err.message); setStatus(null); }
    finally { setLoading(false); }
  }

  useEffect(() => { load(); }, []);

  async function download(project) {
    setDownloadID(project.id); setError('');
    try {
      const response = await fetch(project.downloadUrl);
      if (!response.ok) { const body = await response.json(); throw new Error(body.error || 'Download failed.'); }
      if (!(response.headers.get('content-type') || '').includes('application/zip')) throw new Error('The download was not a ZIP file.');
      const blob = await response.blob(); const href = URL.createObjectURL(blob); const link = document.createElement('a');
      link.href = href; link.download = `${project.id}.zip`; document.body.appendChild(link); link.click(); link.remove(); setTimeout(() => URL.revokeObjectURL(href), 1000);
      setNotice(`Downloaded starter files for ${project.name}.`);
    } catch (err) { setError(err.message); }
    finally { setDownloadID(''); }
  }

  async function create(event) {
    event.preventDefault(); if (!validName || busy) return;
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await requestJSON('/api/projects', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name.trim(), templateId: selected }) });
      setProjects(previous => [result.project, ...previous.filter(p => p.id !== result.project.id)]); setName('');
      setNotice(`${result.project.name} is ready. Your project has been saved.`);
      if (autoDownload) await download(result.project);
    } catch (err) { setError(err.message); }
    finally { setBusy(false); }
  }

  async function more() {
    setBusy(true); setError('');
    try { const data = await requestJSON(`/api/projects?limit=100&cursor=${encodeURIComponent(cursor)}`); setProjects(previous => [...previous, ...data.projects.filter(p => !previous.some(old => old.id === p.id))]); setCursor(data.nextCursor || ''); }
    catch (err) { setError(err.message); } finally { setBusy(false); }
  }

  function choose(templateID) { setSelected(templateID); setView('overview'); setTimeout(() => formRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' }), 50); }
  const selectedTemplate = templates.find(t => t.id === selected);
  const filtered = projects.filter(p => `${p.name} ${p.templateId}`.toLowerCase().includes(search.toLowerCase()));
  const displayed = view === 'overview' ? filtered.slice(0, 5) : filtered;
  const titles = { overview: 'Developer workspace', projects: 'Your projects', templates: 'Project templates' };

  return <div className="app-shell">
    <aside className="sidebar">
      <a className="brand" href="#" onClick={e => { e.preventDefault(); setView('overview'); }}><span className="brand-mark"><Icon name="arrow" size={23}/></span><span>launchpad<span className="brand-caption">DEVELOPER PORTAL</span></span></a>
      <p className="nav-label">WORKSPACE</p>
      <nav aria-label="Main navigation">{[['overview','grid','Overview'],['projects','box','Projects'],['templates','layers','Templates']].map(([id, icon, label]) => <button key={id} className={view === id ? 'nav-item active' : 'nav-item'} aria-current={view === id ? 'page' : undefined} onClick={() => setView(id)}><Icon name={icon}/><span>{label}</span>{id === 'projects' && <span className="nav-count">{projects.length}</span>}</button>)}</nav>
      <div className="sidebar-note"><span className="mini-orbit">↗</span><h3>Less setup. More building.</h3><p>Start with an approved template and make it your own.</p></div>
      <div className="sidebar-footer"><span className="avatar">DEV</span><div><strong>Developer workspace</strong><span>Internal access</span></div><span className="status-dot"/></div>
    </aside>
    <div className="workspace">
      <header className="topbar"><div className="breadcrumb">Workspace <span>/</span> <strong>{view[0].toUpperCase() + view.slice(1)}</strong></div><div className="topbar-right"><span className={`connection ${status ? 'connected' : ''}`}><span className="status-dot"/>{status ? 'Services connected' : loading ? 'Connecting…' : 'Service unavailable'}</span><span className="top-avatar">M</span></div></header>
      <main className="content">
        <div className="page-heading"><div><p className="eyebrow">BUILD WITH CONFIDENCE</p><h1>{titles[view]}</h1><p className="subtitle">{view === 'overview' ? 'Everything you need to start your next project, in one place.' : view === 'projects' ? 'Your generated projects, ready whenever you need them.' : 'Approved starting points for your next application.'}</p></div><button className="button secondary refresh" onClick={load} disabled={loading || busy}><Icon name="refresh"/>{loading ? 'Loading…' : 'Refresh'}</button></div>
        {error && <div className="alert error" role="alert"><span>{error}</span><button aria-label="Dismiss error" onClick={() => setError('')}><Icon name="close"/></button></div>}
        {notice && <div className="alert success" role="status"><Icon name="check"/><span>{notice}</span><button aria-label="Dismiss notification" onClick={() => setNotice('')}><Icon name="close"/></button></div>}
        {view === 'overview' && <>
          <div className="stats-grid"><div className="stat-card"><span className="stat-icon purple"><Icon name="box"/></span><div><span>Projects in your workspace</span><strong>{projects.length}</strong></div><span className="stat-detail">Saved & ready</span></div><div className="stat-card"><span className="stat-icon blue"><Icon name="layers"/></span><div><span>Available templates</span><strong>{templates.length}</strong></div><span className="stat-detail">Approved starters</span></div><div className="stat-card"><span className="stat-icon green"><Icon name="globe"/></span><div><span>Project storage</span><strong className="storage-label">{status?.storage === 'dynamodb' ? 'DynamoDB' : status?.storage === 'file' ? 'Local files' : 'Connecting'}</strong></div><span className="stat-detail">{status ? 'Connected' : 'Waiting for service'}</span></div></div>
          <section className="create-section" ref={formRef} aria-labelledby="create-title"><div className="section-title"><span className="section-icon"><Icon name="plus"/></span><div><h2 id="create-title">Create a new project</h2><p>A working starting point in a few clicks.</p></div><span className="step-label">01 — CHOOSE YOUR STARTER</span></div>
            <form onSubmit={create}><fieldset className="template-options"><legend className="sr-only">Choose a template</legend>{templates.map(t => <label key={t.id} className={`template-option ${selected === t.id ? 'selected' : ''}`}><input type="radio" name="template" value={t.id} checked={selected === t.id} onChange={() => setSelected(t.id)}/><span className={`template-logo ${t.language === 'Go' ? 'go' : 'react'}`}>{t.language === 'Go' ? 'GO' : '⚛'}</span><span className="template-copy"><strong>{t.name}</strong><span>{t.description}</span><span className="template-tags"><em>{t.language}</em><em>{t.files} starter files</em></span></span><span className="radio-indicator">{selected === t.id && <Icon name="check" size={13}/>}</span></label>)}</fieldset>
              <div className="form-bottom"><div className="name-field"><label htmlFor="project-name">Project name</label><input id="project-name" value={name} onChange={e => setName(e.target.value)} placeholder="e.g. payments-api" maxLength={100} required minLength={3} autoComplete="off" aria-describedby="name-help"/><span id="name-help">Use a clear name, between 3 and 50 characters.</span></div><div className="create-controls"><label className="checkbox"><input type="checkbox" checked={autoDownload} onChange={e => setAutoDownload(e.target.checked)}/>Download files after creating</label><button type="submit" className="button primary" disabled={!validName || busy || !status || templates.length === 0}>{busy ? 'Creating project…' : 'Create project'}<Icon name="arrow"/></button></div></div>
            </form><div className="create-footnote"><Icon name="check" size={15}/><span>Includes runnable source code, setup instructions, and a Dockerfile.</span><span className="selected-hint">{selectedTemplate?.language || 'Choose a template'} starter</span></div>
          </section>
        </>}
        {view === 'templates' && <section className="template-gallery">{templates.map(t => <article className="gallery-card" key={t.id}><span className={`template-logo big ${t.language === 'Go' ? 'go' : 'react'}`}>{t.language === 'Go' ? 'GO' : '⚛'}</span><span className="approved"><Icon name="check" size={13}/>Approved</span><h2>{t.name}</h2><p>{t.description}</p><div className="gallery-meta"><span>{t.language}</span><span>v{t.version}</span><span>{t.files} files</span></div><button className="button primary" onClick={() => choose(t.id)}>Use this template<Icon name="arrow"/></button></article>)}</section>}
        {view !== 'templates' && <section className="projects-section"><div className="projects-heading"><div><h2>{view === 'overview' ? 'Recent projects' : 'All projects'}</h2><p>Your starter files are always a download away.</p></div><label className="search-field"><Icon name="search" size={17}/><input aria-label="Search projects" placeholder="Search projects" value={search} onChange={e => setSearch(e.target.value)}/></label></div>
          <div className="project-table"><div className="table-head"><span>PROJECT</span><span>TEMPLATE</span><span>CREATED</span><span>STATUS</span><span/></div>{loading && projects.length === 0 ? <div className="empty-state"><span className="loading-ring"/><h3>Loading your workspace</h3></div> : displayed.length === 0 ? <div className="empty-state"><span className="empty-icon"><Icon name="box" size={28}/></span><h3>{search ? 'No matching projects' : 'Your next project starts here'}</h3><p>{search ? 'Try another name or template.' : 'Choose a template above to create your first project.'}</p>{view === 'projects' && !search && <button className="button secondary" onClick={() => setView('overview')}>Create your first project<Icon name="plus"/></button>}</div> : displayed.map(p => <div className="project-row" key={p.id}><div className="project-title"><span className={`project-icon ${p.templateId === 'go-gin-api' ? 'go' : 'react'}`}><Icon name="code"/></span><div><strong>{p.name}</strong><span>{p.id.slice(0, 20)}…</span></div></div><span className="table-template">{p.templateId === 'go-gin-api' ? 'Go + Gin API' : 'React Frontend'}</span><span className="table-date">{dateLabel(p.createdAt)}</span><span className="ready-pill"><span/>Ready</span><button className="download-button" aria-label={`Download ${p.name}`} disabled={downloadID === p.id} onClick={() => download(p)}><Icon name="download"/><span>{downloadID === p.id ? 'Downloading…' : 'Download'}</span></button></div>)}</div>
          <div className="table-footer"><span>{displayed.length} project{displayed.length === 1 ? '' : 's'} shown</span>{view === 'overview' && projects.length > 5 && <button onClick={() => setView('projects')}>View all projects<Icon name="arrow" size={16}/></button>}{view === 'projects' && cursor && <button disabled={busy} onClick={more}>{busy ? 'Loading…' : 'Load more projects'}<Icon name="arrow" size={16}/></button>}</div>
        </section>}
        <footer className="page-footer"><span>Launchpad · Your developer workspace</span><span><Icon name="check" size={14}/>Build something worth shipping.</span></footer>
      </main>
    </div>
  </div>;
}
