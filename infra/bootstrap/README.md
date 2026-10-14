# Portal operator access

The user approved this setup on 14 October 2026. Terraform applied it successfully: three IAM resources created, with no changes or deletions to existing resources.

It created **portal-operator** in account **316865756899**, with:

- AWS local-development sign-in through the managed `SignInLocalDevelopmentAccess` policy.
- `eks:DescribeCluster` for `developer-portal` in `ap-south-1`.
- Changing its own console password and reading the account password policy.

The main EKS deployment would grant this user Kubernetes administration of the single `developer-portal` practice cluster through an EKS access entry. This operator's AWS policies do not grant account-wide administrator access. The existing `swarnim-admin` user is not changed.

No console password, login profile, or access key is generated here. Set the password directly on AWS:

1. Confirm the console account is **316865756899**.
2. Open **IAM → Users → portal-operator → Security credentials → Console sign-in → Enable console access**.
3. Choose **Custom password** and enter your password on the AWS page.
4. Finish by choosing **Reset password** in the dialog. Wait for the success message confirming that console access is enabled. Opening the dialog alone does not save the password.

Keep the password out of chat and project files. [AWS password setup instructions](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_passwords_admin-change-user.html)

Then sign this user into the local CLI:

```powershell
& '.\.tools\aws\Amazon\AWSCLIV2\aws.exe' login --profile portal-eks --region ap-south-1
```

Select the `portal-operator` IAM user on the AWS sign-in page. The deployment uses `portal` for Terraform and `portal-eks` for kubectl:

```powershell
.\scripts\Deploy-AWS.ps1 -Profile portal -KubernetesProfile portal-eks -Region ap-south-1 -AdminPrincipalArn 'arn:aws:iam::316865756899:user/portal-operator' -AdminCidr 'YOUR_PUBLIC_IP/32' -Apply
```

The script verifies both profiles and the exact operator ARN before deploying. IAM bootstrap and EKS infrastructure keep separate Terraform state files.
