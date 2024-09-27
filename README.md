# aws-lambda-ci-runner
用于golang、npm、yarn、node的CI的aws-lambda执行器，可自行改动
1. 部署到ECR
```
docker build --platform linux/amd64 -t ${aws_account}.dkr.ecr.${region}.amazonaws.com/${repo_name}:${tag} .
docker push ${aws_account}.dkr.ecr.${region}.amazonaws.com/${repo_name}:${tag}
```
2. 创建EFS磁盘，挂载点/mnt/home，预先挂载至CI目标服务器
3. 创建lambda
+ 需挂载EFS的/mnt/home目录
+ 设置Image为ecr的链接：`${aws_account}.dkr.ecr.${region}.amazonaws.com/${repo_name}:${tag}`
+ 环境变量举例：
```GITHUB_PRIVATE_KEY = file("~/.ssh/id_ed25519")
GOLDFLAGS   = "-extldflags --static"
CGO_ENABLED = "1"
GOOS        = "linux"
GOARCH      = "amd64"
HOME = "/mnt/home"
PATH="/usr/local/node/bin:/usr/local/bin:/usr/bin/:/bin:/usr/local/go/bin:/usr/local/sbin:/usr/sbin:/sbin"
```
+ Lambda需要的权限（含EFS）
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
3. 注意事项
+ Dockerfile中golang、node版本可根据实际需求更换
+ 部分前端及node项目磁盘容量不足，需增加额外磁盘，建议至少2G
```
ephemeral_storage {
  size = 2048
}
```
+ efs的相关端口需在安全组中放开
