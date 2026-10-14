param(
  [string]$Profile = 'default',
  [string]$Region = 'ap-south-1',
  [Parameter(Mandatory=$true)][string]$AdminPrincipalArn,
  [Parameter(Mandatory=$true)][string]$AdminCidr,
  [switch]$Apply
)
. "$PSScriptRoot/Common.ps1"
$terraform = Find-PortalTool 'terraform.exe' '.tools/terraform/terraform.exe'
$kubectl = Find-PortalTool 'kubectl.exe' '.tools/kubectl.exe'
$aws = Find-PortalTool 'aws.exe' '.tools/aws/Amazon/AWSCLIV2/aws.exe'
$docker = Find-PortalTool 'docker.exe' '.tools/docker/docker.exe'
$env:AWS_PROFILE = $Profile
$env:AWS_REGION = $Region
$env:AWS_DEFAULT_REGION = $Region
$env:AWS_PAGER = ''
$identityJSON = & $aws sts get-caller-identity --output json
if ($LASTEXITCODE -ne 0) { throw 'AWS sign-in is not available. Sign in with your configured AWS CLI profile first.' }
$identity = $identityJSON | ConvertFrom-Json
if ($identity.Account -ne '316865756899') { throw 'Sign-in belongs to a different AWS account.' }
Invoke-Checked $docker @('info','--format','{{.ServerVersion}}')
$env:TF_VAR_aws_region=$Region
$env:TF_VAR_expected_account_id='316865756899'
$env:TF_VAR_admin_principal_arn=$AdminPrincipalArn
$env:TF_VAR_admin_cidr=$AdminCidr
$infra = Join-Path $PortalRoot 'infra/terraform'
$run = Join-Path $PortalRoot '.run'
New-Item -ItemType Directory -Force $run | Out-Null
Push-Location $infra
try {
  Invoke-Checked $terraform @('init','-input=false')
  Invoke-Checked $terraform @('validate')
  Invoke-Checked $terraform @('plan','-input=false','-out',"$run/aws.tfplan")
  if (-not $Apply) { Write-Host 'Plan saved. Add -Apply to deploy this configuration. EKS resources incur ongoing AWS charges.'; return }
  Invoke-Checked $terraform @('apply','-input=false',"$run/aws.tfplan")
  $outputJSON = & $terraform output -json
  if ($LASTEXITCODE -ne 0) { throw 'Could not read Terraform outputs.' }
  $outputs = $outputJSON | ConvertFrom-Json
} finally { Pop-Location }
$tag = 'release-' + (Get-Date).ToUniversalTime().ToString('yyyyMMddHHmmss')
$repositories = $outputs.repositories.value
$images = @{}
foreach ($service in @('portal-api','project-service','frontend')) {
  $images[$service] = $repositories.$service + ':' + $tag
  if ($service -eq 'frontend') { Invoke-Checked $docker @('build','--platform','linux/amd64','-t',$images[$service],"$PortalRoot/frontend") }
  else { Invoke-Checked $docker @('build','--platform','linux/amd64','--target',$service,'-t',$images[$service],"$PortalRoot/backend/portal-api") }
}
$registry = ($repositories.frontend -split '/')[0]
# The token is sent directly to docker; it is never written to a file or printed.
& $aws ecr get-login-password --region $Region | & $docker login --username AWS --password-stdin $registry
if ($LASTEXITCODE -ne 0) { throw 'ECR login failed.' }
foreach ($service in @('portal-api','project-service','frontend')) { Invoke-Checked $docker @('push',$images[$service]) }
$kubeconfig = Join-Path $run 'eks-kubeconfig'
Invoke-Checked $aws @('eks','update-kubeconfig','--name',$outputs.cluster_name.value,'--region',$Region,'--profile',$Profile,'--kubeconfig',$kubeconfig)
$manifest = [IO.File]::ReadAllText((Join-Path $PortalRoot 'infra/kubernetes/portal.yaml.tmpl'))
$values = @{ '__PROJECT_ROLE_ARN__'=$outputs.project_role_arn.value; '__AWS_REGION__'=$Region; '__TABLE_NAME__'=$outputs.table_name.value; '__PORTAL_IMAGE__'=$images['portal-api']; '__PROJECT_IMAGE__'=$images['project-service']; '__FRONTEND_IMAGE__'=$images['frontend'] }
foreach ($key in $values.Keys) { $manifest = $manifest.Replace($key,$values[$key]) }
if ($manifest -match '__[A-Z_]+__') { throw 'An unresolved placeholder remains in the Kubernetes manifest.' }
$manifestPath = Join-Path $run 'portal-eks.yaml'
[IO.File]::WriteAllText($manifestPath,$manifest)
Invoke-Checked $kubectl @('--kubeconfig',$kubeconfig,'apply','-f',$manifestPath)
foreach ($service in @('project-service','portal-api','frontend')) { Invoke-Checked $kubectl @('--kubeconfig',$kubeconfig,'-n','portal','rollout','status',"deployment/$service",'--timeout=300s') }
Write-Host 'Deployments are ready. Open a separate terminal and run:'
Write-Host "kubectl --kubeconfig `"$kubeconfig`" -n portal port-forward service/frontend 8090:8080 --address 127.0.0.1"
Write-Host 'Then open http://127.0.0.1:8090 and run frontend browser checks with PORTAL_TEST_URL set to that address.'
