#!/usr/bin/env python3
import os

import aws_cdk as cdk
from dotenv import load_dotenv

from home_it_services.qr_generator_stack import QRGeneratorStack

load_dotenv()
hosted_zone_id = os.getenv("HOSTED_ZONE_ID")
zone_name = os.getenv("ZONE_NAME")
qr_subdomain = os.getenv("QR_SUBDOMAIN")
qr_ses_recipient = os.getenv("QR_SES_RECIPIENT")
qr_ses_identity = os.getenv("QR_SES_IDENTITY")

app = cdk.App()
env = cdk.Environment(account=os.getenv('CDK_DEFAULT_ACCOUNT'), region=os.getenv('CDK_DEFAULT_REGION'))
QRGeneratorStack(app, "QRGeneratorStack", hosted_zone_id, zone_name, qr_subdomain, qr_ses_recipient, qr_ses_identity, env=env)

app.synth()
