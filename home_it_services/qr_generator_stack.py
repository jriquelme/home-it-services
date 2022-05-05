from aws_cdk import (
    RemovalPolicy,
    Stack,
    aws_s3 as s3,
    aws_s3_deployment as s3_deployment,
    aws_route53 as route53,
    aws_route53_targets as route53_targets,
    aws_ses as ses,
    aws_ses_actions as ses_actions,
    aws_sns as sns,
    aws_sns_subscriptions as sns_subscriptions,
    aws_lambda_go_alpha as lambda_go,
)
from constructs import Construct


class QRGeneratorStack(Stack):

    def __init__(self, scope: Construct, construct_id: str, hosted_zone_id: str, zone_name: str, subdomain: str,
                 ses_recipient: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # configure email receiving (the domain must be properly configured in SES. Please read
        # https://docs.aws.amazon.com/ses/latest/dg/receiving-email.html)
        emails = s3.Bucket(self, "Emails", auto_delete_objects=True, block_public_access=s3.BlockPublicAccess.BLOCK_ALL,
                           removal_policy=RemovalPolicy.DESTROY)
        notifications = sns.Topic(self, "Notifications")
        # the RuleSet default-rule-set must exist and be the active one
        rule_set = ses.ReceiptRuleSet.from_receipt_rule_set_name(self, "RuleSet",
                                                                 receipt_rule_set_name="default-rule-set")
        rule_set.add_rule("QREmail", actions=[ses_actions.S3(bucket=emails, topic=notifications)],
                          recipients=[ses_recipient])

        # bucket to store files
        files = self.website_bucket(hosted_zone_id, zone_name, subdomain)

        # lambda to process notifications
        qr_app = lambda_go.GoFunction(self, "QRApp", entry="qrapp",
                                      bundling=lambda_go.BundlingOptions(
                                          environment={
                                              "FILES_BUCKET": files.bucket_name,
                                          },
                                          go_build_flags=["-ldflags \"-s -w\""]))
        notifications.add_subscription(sns_subscriptions.LambdaSubscription(qr_app))
        emails.grant_read_write(qr_app.role)
        files.grant_read_write(qr_app.role)

    def website_bucket(self, hosted_zone_id, zone_name, subdomain):
        # bucket to store the uploaded files
        bucket_name = f"{subdomain}.{zone_name}"
        files = s3.Bucket(self, "Files", bucket_name=bucket_name, public_read_access=True,
                          website_error_document="error.html", website_index_document="index.html")
        deployment = s3_deployment.BucketDeployment(self, "DeployFiles", destination_bucket=files,
                                                    sources=[s3_deployment.Source.asset("qr_website")])
        # expose files at a subdomain
        hosted_zone = route53.PublicHostedZone.from_hosted_zone_attributes(self, "Domain",
                                                                           hosted_zone_id=hosted_zone_id,
                                                                           zone_name=zone_name)
        route53.ARecord(self, "QRSubdomain",
                        target=route53.RecordTarget.from_alias(route53_targets.BucketWebsiteTarget(files)),
                        zone=hosted_zone, record_name=subdomain)
        return files
