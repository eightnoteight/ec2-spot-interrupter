package main

import (
	"context"
	"flag"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/fis"
	"github.com/aws/aws-sdk-go-v2/service/fis/types"
	"github.com/sirupsen/logrus"
)

var (
	logger = logrus.New()
)

func createExperimentTemplate(cfg aws.Config, instanceArn string, fisRoleArn string) (string, error) {

	client := fis.NewFromConfig(cfg)
	input := &fis.CreateExperimentTemplateInput{
		Description: aws.String("Created by ec2-spot-interrupter tool"),
		StopConditions: []types.CreateExperimentTemplateStopConditionInput{
			{
				Source: aws.String("none"),
			},
		},
		Targets: map[string]types.CreateExperimentTemplateTargetInput{
			"targetSpot": {
				ResourceType:  aws.String("aws:ec2:spot-instance"),
				ResourceArns:  []string{instanceArn},
				SelectionMode: aws.String("ALL"),
			},
		},
		Actions: map[string]types.CreateExperimentTemplateActionInput{
			"actionSpot": {
				ActionId: aws.String("aws:ec2:send-spot-instance-interruptions"),
				Parameters: map[string]string{
					"durationBeforeInterruption": "PT0M",
				},
				Targets: map[string]string{
					"SpotInstances": "targetSpot",
				},
			},
		},
		RoleArn: aws.String(fisRoleArn),
	}

	resp, err := client.CreateExperimentTemplate(context.Background(), input)
	if err != nil {
		return "", err
	}

	return *resp.ExperimentTemplate.Id, nil
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		logger.Fatalf("Configuration error: %v", err)
	}

	// Define command-line flags
	createTemplateFlag := flag.Bool("create-template", false, "Create an experiment template")
	interruptSpotFlag := flag.Bool("interrupt-spot", false, "Interrupt a spot instance")
	templateIDFlag := flag.String("template-id", "", "Template ID for interrupting a spot instance")
	instanceARNFlag := flag.String("instance-arn", "", "Instance ARN for creating a template")
	fisRoleARNFlag := flag.String("fis-role-arn", "", "FIS role ARN for creating a template")

	// Parse command-line flags
	flag.Parse()

	instanceArn := *instanceARNFlag
	fisRoleArn := *fisRoleARNFlag
	createTemplate := *createTemplateFlag // Get the value of the createTemplate flag
	interruptSpot := *interruptSpotFlag   // Get the value of the interruptSpot flag
	templateID := *templateIDFlag         // Get the value of the templateID flag

	if createTemplate && interruptSpot {
		logger.Fatal("Both --create-template and --interrupt-spot flags are specified. Please specify only one.")
	}

	if createTemplate {
		templateID, err := createExperimentTemplate(cfg, instanceArn, fisRoleArn)
		if err != nil {
			logger.Fatalf("Failed to create template: %v", err)
		}
		logger.Printf("Created template with ID: %s", templateID)
	} else if interruptSpot {
		if templateID == "" {
			logger.Fatal("templateID flag must be specified when using interruptSpot.")
		}

		client := fis.NewFromConfig(cfg)
		ctx := context.Background()

		_, err = client.StartExperiment(ctx, &fis.StartExperimentInput{
			ExperimentTemplateId: aws.String(templateID),
		})
		if err != nil {
			logger.Fatalf("Failed to trigger experiment: %v", err)
		}
		logger.Print("Successfully triggered experiment")
	} else {
		logger.Fatal("Either --create-template or --interrupt-spot flag must be specified.")
	}
}
