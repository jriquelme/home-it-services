from aws_cdk import (
    RemovalPolicy,
    Stack,
    aws_s3 as s3,
    aws_s3_deployment as s3_deployment,
    aws_route53 as route53,
    aws_route53_targets as route53_targets,
)
from constructs import Construct


class QRWebsiteStack(Stack):
    @property
    def files_bucket(self):
        return self._files_bucket

    def __init__(self, scope: Construct, construct_id: str, hosted_zone_id: str, zone_name: str, subdomain: str,
                 **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # bucket to store the uploaded files
        bucket_name = f"{subdomain}.{zone_name}"
        files = s3.Bucket(self, "Files", auto_delete_objects=True, bucket_name=bucket_name, public_read_access=True,
                          removal_policy=RemovalPolicy.DESTROY, website_error_document="error.html",
                          website_index_document="index.html")
        deployment = s3_deployment.BucketDeployment(self, "DeployFiles", destination_bucket=files,
                                                    sources=[s3_deployment.Source.asset("qr_website")])
        # expose files at a subdomain
        hosted_zone = route53.PublicHostedZone.from_hosted_zone_attributes(self, "Domain",
                                                                           hosted_zone_id=hosted_zone_id,
                                                                           zone_name=zone_name)
        route53.ARecord(self, "QRSubdomain",
                        target=route53.RecordTarget.from_alias(route53_targets.BucketWebsiteTarget(files)),
                        zone=hosted_zone, record_name=subdomain)

        self._files_bucket = files
