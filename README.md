# Home IT services

This project showcases simple serverless solutions using AWS services. It's mainly a project to experiment with
[CDK](https://docs.aws.amazon.com/cdk/index.html) (using as an excuse the *requirements* at home like *"please make me
a QR code"* :sweat_smile:)

## QR Generator

Receive an email and publish the attachments to a static website, generating a QR code for each file using the destination
URL. The URL of the attachment file and the QR code are returned in a response email.

### Requirements

* A Hosted Zone in Route53 (e.x: yourdomain.com)
* A SES verified identify to receive emails, properly configured in Route53 (e.x: mail.yourdomain.com)

### Resources

* error.html taken from [https://codepen.io/jkantner/pen/aPLWJm](https://codepen.io/jkantner/pen/aPLWJm)
* QR codes are generated with [github.com/yeqown/go-qrcode](https://github.com/yeqown/go-qrcode)

## Deploying

This a python CDK project, perform the usual spells:

```
$ python3 -m venv .venv
$ source .venv/bin/activate
$ pip install -r requirements.txt
```

Make an `.env` file using `.env.example` as template. Deploy with:

```
$ cdk deploy
```
