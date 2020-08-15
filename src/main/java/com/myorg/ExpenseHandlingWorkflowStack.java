package com.myorg;

import java.util.HashMap;
import java.util.List;
import java.util.Collections;

import software.amazon.awscdk.core.Stack;
import software.amazon.awscdk.core.Duration;
import software.amazon.awscdk.core.Construct;
import software.amazon.awscdk.core.CfnParameter;
import software.amazon.awscdk.core.CfnParameterProps;

import software.amazon.awscdk.services.stepfunctions.Chain;
import software.amazon.awscdk.services.stepfunctions.Choice;
import software.amazon.awscdk.services.stepfunctions.Condition;
import software.amazon.awscdk.services.stepfunctions.Context;
import software.amazon.awscdk.services.stepfunctions.IntegrationPattern;
import software.amazon.awscdk.services.stepfunctions.StateMachine;
import software.amazon.awscdk.services.stepfunctions.TaskInput;
import software.amazon.awscdk.services.stepfunctions.tasks.LambdaInvoke;
import software.amazon.awscdk.services.stepfunctions.tasks.LambdaInvokeProps;

import software.amazon.awscdk.services.lambda.Code;
import software.amazon.awscdk.services.lambda.Runtime;
import software.amazon.awscdk.services.lambda.Function;

import software.amazon.awscdk.services.s3.Bucket;

import software.amazon.awscdk.services.dynamodb.Table;
import software.amazon.awscdk.services.dynamodb.Attribute;
import software.amazon.awscdk.services.dynamodb.TableProps;
import software.amazon.awscdk.services.dynamodb.AttributeType;

import software.amazon.awscdk.services.sns.Topic;
import software.amazon.awscdk.services.sns.TopicProps;
import software.amazon.awscdk.services.sns.subscriptions.EmailSubscription;

import software.amazon.awscdk.services.apigateway.AuthorizationType;
import software.amazon.awscdk.services.apigateway.Deployment;
import software.amazon.awscdk.services.apigateway.DeploymentProps;
import software.amazon.awscdk.services.apigateway.IntegrationResponse;
import software.amazon.awscdk.services.apigateway.LambdaIntegration;
import software.amazon.awscdk.services.apigateway.LambdaIntegrationOptions;
import software.amazon.awscdk.services.apigateway.Method;
import software.amazon.awscdk.services.apigateway.MethodOptions;
import software.amazon.awscdk.services.apigateway.MethodResponse;
import software.amazon.awscdk.services.apigateway.Resource;
import software.amazon.awscdk.services.apigateway.RestApi;
import software.amazon.awscdk.services.apigateway.RestApiProps;
import software.amazon.awscdk.services.apigateway.Stage;
import software.amazon.awscdk.services.apigateway.StageProps;

import software.amazon.awscdk.services.cloudtrail.Trail;
import software.amazon.awscdk.services.cloudtrail.TrailProps;
import software.amazon.awscdk.services.cloudtrail.S3EventSelector;

import software.amazon.awscdk.services.events.Rule;
import software.amazon.awscdk.services.events.RuleProps;
import software.amazon.awscdk.services.events.EventPattern;
import software.amazon.awscdk.services.events.targets.SfnStateMachine;
import software.amazon.awscdk.services.events.targets.SfnStateMachineProps;
import software.amazon.awscdk.services.iam.Effect;
import software.amazon.awscdk.services.iam.PolicyStatement;
import software.amazon.awscdk.services.iam.PolicyStatementProps;


public class ExpenseHandlingWorkflowStack extends Stack {

    public ExpenseHandlingWorkflowStack(final Construct scope, final String id) {
        super(scope, id);

        try {

            /** CONSTRUCS */

            // Process Receipts (bucket, function, privileges to use bucket)
            Bucket expensesBucket = new Bucket(this, "ExpensesRepository");

            Function processReceipt = Function.Builder.create(this, "ProcessReceiptHandler").runtime(Runtime.GO_1_X)
                    .code(Code.fromAsset("resources/receipt/bin")).timeout(Duration.seconds(120)).handler("processReceipt").build();

            expensesBucket.grantRead(processReceipt);
            processReceipt.addToRolePolicy(new PolicyStatement(PolicyStatementProps.builder()
                                                   .actions(Collections.singletonList("textract:*"))
                                                   .resources(Collections.singletonList("*"))
                                                   .effect(Effect.ALLOW)
                                                   .build()));

            // Notify Submitter (topic, subscriber, function, privileges to use topic)
            Topic submitterTopic = new Topic(this, "ExpenseStatusNotificationTopic", TopicProps.builder().build());

            CfnParameter submitter = new CfnParameter(this, "SubscribedExpenseSubmitter",
                    CfnParameterProps.builder().type("String").defaultValue("email-alias").build());

            submitterTopic.addSubscription(new EmailSubscription(submitter.getValueAsString() + "@email-domain.extension"));

            Function notifySubmitter = Function.Builder.create(this, "NotifySubmitterHandler").runtime(Runtime.GO_1_X)
                    .code(Code.fromAsset("resources/notification/bin")).handler("notifySubmitter")
                    .environment(new HashMap<String, String>() {{ put("TOPIC", submitterTopic.getTopicArn()); }})
                    .build();

            submitterTopic.grantPublish(notifySubmitter);


            // Register Expense (dynamodb table, function, privileges to use dynamodb)
            Table expenses = new Table(this, "Expenses",
                    TableProps.builder()
                            .partitionKey(Attribute.builder().name("ExpenseID").type(AttributeType.STRING).build())
                            .writeCapacity(4).readCapacity(4).build());

            Function registerExpense = Function.Builder.create(this, "RegisterExpenseHandler").runtime(Runtime.GO_1_X)
                    .code(Code.fromAsset("resources/expense/bin")).timeout(Duration.seconds(120)).handler("registerExpense")
                    .environment(new HashMap<String, String>() {{ put("TABLE", expenses.getTableName()); }})
                    .build();

            expenses.grantReadWriteData(registerExpense);



            // Handle Manual Process (function, approval topic?, subscriber, privileges to use that topic)
            
            Topic approverTopic = new Topic(this, "ExpenseApproverTopic", TopicProps.builder().build());

            CfnParameter approver = new CfnParameter(this, "SubscribedApprover",
                    CfnParameterProps.builder().type("String").defaultValue("email-alias").build());

            approverTopic.addSubscription(new EmailSubscription(approver.getValueAsString() + "@email-domain.extension"));

            Function requestApproval = Function.Builder.create(this, "ManualApproveProcessHandler").runtime(Runtime.GO_1_X)
                    .code(Code.fromAsset("resources/request/bin")).handler("requestApproval")
                    .environment(new HashMap<String, String>() {{ put("TOPIC", approverTopic.getTopicArn()); }})
                    .build();

            approverTopic.grantPublish(requestApproval);

            Function sendApproval = Function.Builder.create(this, "SendApprovalHandler").runtime(Runtime.GO_1_X)
                    .code(Code.fromAsset("resources/process/bin")).timeout(Duration.seconds(120)).handler("processApproval")
                    .build();

            sendApproval.addToRolePolicy(new PolicyStatement(PolicyStatementProps.builder()
                                                   .actions(Collections.unmodifiableList(List.of("states:SendTaskSuccess", "states:SendTaskFailure")))
                                                   .resources(Collections.singletonList("*"))
                                                   .effect(Effect.ALLOW)
                                                   .build()));


            RestApi approvalProcessApi = new RestApi(this, "manual-approve-process-api", RestApiProps.builder()
                      .deploy(true).restApiName("ManualApprovalProcessApi")
                      .description("Endpoint for approving expense requests")
                      .defaultMethodOptions(MethodOptions.builder()
                              .authorizationType(AuthorizationType.NONE)
                              .build())
                      .deploy(false)
                      .build());

            Deployment deployment = new Deployment(this, "manual-approve-process-api-deployment", DeploymentProps.builder()
                      .api(approvalProcessApi)
                      .build());

            Stage stage = new Stage(this, "manual-approve-process-api-stage", StageProps.builder()
                      .stageName("states")
                      .deployment(deployment)
                      .build());

            approvalProcessApi.setDeploymentStage(stage);

            Resource approvalResource = approvalProcessApi.getRoot().addResource("execution");
            Method getApprovalStatus = approvalResource.addMethod("GET", new LambdaIntegration(sendApproval, 
                    LambdaIntegrationOptions.builder().proxy(false).integrationResponses(Collections.singletonList(IntegrationResponse.builder()
                                                  .statusCode("200")
                                                  .build()))
                          .requestTemplates(new HashMap<String, String>() {{
                                  put("application/json",
                                  "{\n" +
                                  " \"body\" : $input.json('$'),\n" +
                                  " \"query\": {\n" +
                                  "    #foreach($queryParam in $input.params().querystring.keySet())\n" +
                                  "    \"$queryParam\": \"$util.escapeJavaScript($input.params().querystring.get($queryParam))\" #if($foreach.hasNext),#end\n" +
                                  "    \n" +
                                  "    #end\n" +
                                  "  }\n" +  
                                  "}");
                                }})
                          .build()),
                          MethodOptions.builder().methodResponses(Collections.singletonList(MethodResponse.builder()
                                      .statusCode("200")
                                      .build())).build()
                          );


            // Approve Expense (function, privileges to write to the dynamo table)
            Function approveExpense = Function.Builder.create(this, "ApproveExpenseHandler").runtime(Runtime.GO_1_X)
                    .code(Code.fromAsset("resources/approval/bin")).handler("approveExpense")
                    .environment(new HashMap<String, String>() {{put("TABLE", expenses.getTableName());}})
                    .build();

            expenses.grantReadWriteData(approveExpense);

            // Extra constructs for starting the process on an expense submission event (cloudtrail, cloudwatch rule)
            Bucket trailLogsBucket = new Bucket(this, "WorkflowCloudTrailBucket");

            Trail receiptSubmissionTrail = new Trail(this, "ReceiptSubmissionCloudTrail", TrailProps.builder()
                    .trailName("ReceiptsSubmossionTrail")
                    .bucket(trailLogsBucket)
                    .build());

            receiptSubmissionTrail.addS3EventSelector(Collections.singletonList(S3EventSelector.builder().bucket(expensesBucket).build()));


            

            /** STATES */

            // Process Receipts Task State
            LambdaInvoke processReceiptTask = new LambdaInvoke(this, "Process Receipt", LambdaInvokeProps.builder()
                    .lambdaFunction(processReceipt).resultPath("$.result").outputPath("$").build());

            // Receipt Validity Choice State
            Choice isReceiptValid = Choice.Builder.create(this, "Is Receipt Valid?").build();

            // Notify Submitter Task State
            LambdaInvoke notifySubmitterTask = new LambdaInvoke(this, "Send Alert to Submitter", LambdaInvokeProps.builder()
                    .lambdaFunction(notifySubmitter).resultPath("$.result").outputPath("$").build());

            // Register Expense Task State
            LambdaInvoke registerExpenseTask = new LambdaInvoke(this, "Register Expense", LambdaInvokeProps.builder()
                    .lambdaFunction(registerExpense).resultPath("$.result").outputPath("$").build());

            // Autoapproval Choice State (less than 50$)
            Choice isExpenseLessThan50 = Choice.Builder.create(this, "Is Expense Less Than 50$?").build();            

            // Kickstart Manual Approval Process Task State
            LambdaInvoke requestApprovalTask = new LambdaInvoke(this, "Request Approval", LambdaInvokeProps.builder()
                    .integrationPattern(IntegrationPattern.WAIT_FOR_TASK_TOKEN)
                    .payload(TaskInput.fromObject(new HashMap<String, Object>() {{
                            put("input.$", "$"); 
                            put("taskToken", Context.getTaskToken());
                            put("ExecutionContext.$", "$$");
                            put("APIGatewayEndpoint", approvalProcessApi.getUrl());
                        }}))
                    .heartbeat(Duration.seconds(600))
                    .lambdaFunction(requestApproval).inputPath("$").resultPath("$.result").outputPath("$").build());

            // Expense Approved Choice State
            Choice isExpenseApproved = Choice.Builder.create(this, "Is Expense Approved?").inputPath("$").outputPath("$") .build();               

            // Approve Expense Task State
            LambdaInvoke approveExpenseTask = new LambdaInvoke(this, "Approve Expense", LambdaInvokeProps.builder()
                    .lambdaFunction(approveExpense).resultPath("$.result").outputPath("$").build());



            /** STATE MACHINE */

            Chain chain = Chain.start(processReceiptTask)
                               .next(isReceiptValid
                                     .when(Condition.stringEquals("$.result.Payload.processReceiptTaskStatus", "Success"), registerExpenseTask
                                     .next(isExpenseLessThan50
                                           .when(Condition.stringEquals("$.result.Payload.registerExpenseTaskStatus", "<=50"), approveExpenseTask)
                                           .when(Condition.stringEquals("$.result.Payload.registerExpenseTaskStatus", ">50"), requestApprovalTask
                                           .next(isExpenseApproved
                                                 .when(Condition.stringEquals("$.result.expenseStatus", "Expense is approved"), approveExpenseTask)
                                                 .when(Condition.stringEquals("$.result.expenseStatus", "Expense is rejected"), notifySubmitterTask)))))
                                     .when(Condition.stringEquals("$.result.Payload.processReceiptTaskStatus", "Failure"), notifySubmitterTask));

            StateMachine expenseWorkflow = StateMachine.Builder.create(this, "ExpenseWorkflow").definition(chain).build();

            
            // CloudWatch event rule for incoming receipts in the receipts S3 bucket causing the state machine to start
            new Rule(this, "PutObjectEventRule", RuleProps.builder()
                  .enabled(true)
                  .ruleName("ReceiptsBucketPutObjectRule")
                  .eventPattern(EventPattern.builder()
                      .source(Collections.singletonList("aws.s3"))
                      .detail(new HashMap<String, Object>() {{
                           put("eventName", Collections.singletonList("PutObject"));
                           put("requestParameters", Collections.singletonMap("bucketName", Collections.singletonList(expensesBucket.getBucketName())));
                      }})
                      .build())
                  .targets(Collections.singletonList(new SfnStateMachine(expenseWorkflow, SfnStateMachineProps.builder().build())))
                  .build());


        } catch (Exception e) {
            e.printStackTrace();
        }
    }

}
