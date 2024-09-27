# aws-lambda-ci-runner
用于golang、npm、yarn、node的CI的aws-lambda执行器，可自行改动
1. Dockerfile中golang、node版本可根据实际需求更换
2. 建议使用EFS磁盘，lambda和CI的目标服务器都绑定其$HOME目录，可以多处绑定以便进行资源共享
3. 环境变量举例：
```GITHUB_PRIVATE_KEY = file("~/.ssh/id_ed25519")
GOLDFLAGS   = "-extldflags --static"
CGO_ENABLED = "1"
GOOS        = "linux"
GOARCH      = "amd64"
HOME = "/mnt/home"
PATH="/usr/local/node/bin:/usr/local/bin:/usr/bin/:/bin:/usr/local/go/bin:/usr/local/sbin:/usr/sbin:/sbin"
```
4. Lambda需要的权限（含EFS）
```
ec2:AllocateAddress
ec2:AssociateAddress
ec2:DescribeAddresses
ec2:DescribeNetworkInterfaces
ec2:CreateNetworkInterface
ec2:DeleteNetworkInterface
ec2:DescribeSubnets
ec2:DescribeSecurityGroups
ec2:DescribeVpcs
ec2:CreateSecurityGroup
ec2:AuthorizeSecurityGroupIngress
ec2:AuthorizeSecurityGroupEgress
ec2:RevokeSecurityGroupIngress
ec2:RevokeSecurityGroupEgress
elasticfilesystem:CreateFileSystem
elasticfilesystem:DescribeFileSystems
elasticfilesystem:DeleteFileSystem
elasticfilesystem:CreateMountTarget
elasticfilesystem:DescribeMountTargets
elasticfilesystem:DeleteMountTarget
elasticfilesystem:CreateAccessPoint
elasticfilesystem:DescribeAccessPoints
elasticfilesystem:DeleteAccessPoint
elasticfilesystem:UpdateFileSystem
elasticfilesystem:UpdateMountTarget
elasticfilesystem:UpdateAccessPoint
lambda:CreateFunction
lambda:UpdateFunctionCode
lambda:GetFunction
lambda:InvokeFunction
lambda:CreateFunctionUrlConfig
lambda:GetFunctionUrlConfig
lambda:UpdateFunctionUrlConfig
lambda:DeleteFunction
lambda:DeleteFunctionUrlConfig
iam:CreateRole
iam:AttachRolePolicy
iam:CreatePolicy
iam:PassRole
iam:PutRolePolicy
iam:DeleteRolePolicy
iam:DetachRolePolicy
iam:DeleteRole
logs:CreateLogGroup
logs:CreateLogStream
logs:PutLogEvents
logs:DescribeLogStreams
logs:DescribeLogGroup
ecr:CreateRepository
ecr:DescribeRepositories
ecr:BatchCheckLayerAvailability
ecr:PutImage
ecr:UploadLayerPart
ecr:CompleteLayerUpload
ecr:InitiateLayerUpload
ecr:BatchGetImage
ecr:ListImages
ecr:SetRepositoryPolicy
ecr:GetDownloadUrlForLayer
ecr:PutLifecyclePolicy
```
```
sts:AssumeRole lambda.amazonaws.com
```
6. 部分前端及node项目磁盘容量不足，需增加额外磁盘，建议至少2G
```
ephemeral_storage {
  size = 2048
}
```
7. efs的相关端口需在安全组中放开
