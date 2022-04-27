import aws_cdk as core
import aws_cdk.assertions as assertions

from home_it_services.home_it_services_stack import HomeItServicesStack

# example tests. To run these tests, uncomment this file along with the example
# resource in home_it_services/home_it_services_stack.py
def test_sqs_queue_created():
    app = core.App()
    stack = HomeItServicesStack(app, "home-it-services")
    template = assertions.Template.from_stack(stack)

#     template.has_resource_properties("AWS::SQS::Queue", {
#         "VisibilityTimeout": 300
#     })
