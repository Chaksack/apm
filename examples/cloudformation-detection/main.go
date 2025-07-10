package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ybke/apm/pkg/cloud"
)

func main() {
	ctx := context.Background()

	// Create AWS provider
	provider, err := cloud.NewAWSProvider(nil)
	if err != nil {
		log.Fatalf("Failed to create AWS provider: %v", err)
	}

	fmt.Println("🔍 CloudFormation Stack Detection for APM Infrastructure")
	fmt.Println("========================================================")

	// Test 1: List all CloudFormation stacks
	fmt.Println("\n1. Listing CloudFormation stacks...")
	stacks, err := provider.ListCloudFormationStacks(ctx, &cloud.StackFilters{
		Regions: []string{"us-east-1", "us-west-2"},
		StackStatus: []string{
			"CREATE_COMPLETE",
			"UPDATE_COMPLETE",
			"UPDATE_ROLLBACK_COMPLETE",
		},
	})
	if err != nil {
		log.Printf("Error listing stacks: %v", err)
	} else {
		fmt.Printf("Found %d CloudFormation stacks\n", len(stacks))
		for _, stack := range stacks {
			fmt.Printf("  - %s (%s) in %s - Status: %s\n", 
				stack.Name, stack.Arn, stack.Region, stack.Status)
		}
	}

	// Test 2: List APM-specific stacks
	fmt.Println("\n2. Listing APM-specific CloudFormation stacks...")
	apmStacks, err := provider.ListAPMStacks(ctx, []string{"us-east-1", "us-west-2"})
	if err != nil {
		log.Printf("Error listing APM stacks: %v", err)
	} else {
		fmt.Printf("Found %d APM-related stacks\n", len(apmStacks))
		for _, stack := range apmStacks {
			fmt.Printf("  - %s in %s - APM Resources: %v\n", 
				stack.Name, stack.Region, stack.APMResources != nil)
			
			if stack.APMResources != nil {
				printAPMResources(stack.APMResources)
			}
		}
	}

	// Test 3: Get detailed stack information if we have stacks
	if len(stacks) > 0 {
		fmt.Println("\n3. Getting detailed information for first stack...")
		firstStack := stacks[0]
		
		detailedStack, err := provider.GetCloudFormationStack(ctx, firstStack.Name, firstStack.Region)
		if err != nil {
			log.Printf("Error getting stack details: %v", err)
		} else {
			fmt.Printf("Stack: %s\n", detailedStack.Name)
			fmt.Printf("  Status: %s\n", detailedStack.Status)
			fmt.Printf("  Created: %s\n", detailedStack.CreatedTime.Format(time.RFC3339))
			fmt.Printf("  Resources: %d\n", len(detailedStack.Resources))
			fmt.Printf("  Parameters: %d\n", len(detailedStack.Parameters))
			fmt.Printf("  Outputs: %d\n", len(detailedStack.Outputs))
			fmt.Printf("  Tags: %d\n", len(detailedStack.Tags))
			fmt.Printf("  Is APM Stack: %v\n", detailedStack.IsAPMStack)

			// Print some resources
			fmt.Println("  Top 5 Resources:")
			for i, resource := range detailedStack.Resources {
				if i >= 5 {
					break
				}
				fmt.Printf("    - %s (%s): %s\n", 
					resource.LogicalID, resource.Type, resource.Status)
			}

			// Print outputs if any
			if len(detailedStack.Outputs) > 0 {
				fmt.Println("  Outputs:")
				for key, value := range detailedStack.Outputs {
					fmt.Printf("    %s: %s\n", key, value)
				}
			}
		}

		// Test 4: Check stack health
		fmt.Println("\n4. Validating stack health...")
		health, err := provider.ValidateCloudFormationStackHealth(ctx, firstStack.Name, firstStack.Region)
		if err != nil {
			log.Printf("Error validating stack health: %v", err)
		} else {
			fmt.Printf("Overall Health: %s\n", health.OverallHealth)
			fmt.Printf("Healthy Resources: %d\n", health.HealthyResources)
			fmt.Printf("Unhealthy Resources: %d\n", health.UnhealthyResources)
			
			if len(health.Issues) > 0 {
				fmt.Println("Issues:")
				for _, issue := range health.Issues {
					fmt.Printf("  - %s\n", issue)
				}
			}

			if len(health.Recommendations) > 0 {
				fmt.Println("Recommendations:")
				for _, rec := range health.Recommendations {
					fmt.Printf("  - %s\n", rec)
				}
			}
		}

		// Test 5: Detect drift (this might take a while)
		fmt.Println("\n5. Detecting stack drift (this may take a few minutes)...")
		drift, err := provider.DetectCloudFormationStackDrift(ctx, firstStack.Name, firstStack.Region)
		if err != nil {
			log.Printf("Error detecting drift: %v", err)
		} else {
			fmt.Printf("Drift Status: %s\n", drift.DriftStatus)
			fmt.Printf("Total Resources: %d\n", drift.TotalResources)
			fmt.Printf("Drifted Resources: %d\n", drift.DriftedCount)
			fmt.Printf("Detection Time: %s\n", drift.DetectionTime.Format(time.RFC3339))

			if len(drift.DriftedResources) > 0 {
				fmt.Println("Drifted Resources:")
				for i, drifted := range drift.DriftedResources {
					if i >= 3 { // Show only first 3
						fmt.Printf("  ... and %d more\n", len(drift.DriftedResources)-3)
						break
					}
					fmt.Printf("  - %s (%s): %s\n", 
						drifted.LogicalID, drifted.ResourceType, drifted.DriftStatus)
				}
			}

			if len(drift.RecommendedActions) > 0 {
				fmt.Println("Recommended Actions:")
				for _, action := range drift.RecommendedActions {
					fmt.Printf("  - %s\n", action)
				}
			}
		}
	}

	// Test 6: Get APM infrastructure summary
	fmt.Println("\n6. Getting APM infrastructure summary...")
	summary, err := provider.GetAPMStackSummary(ctx, []string{"us-east-1", "us-west-2"})
	if err != nil {
		log.Printf("Error getting APM summary: %v", err)
	} else {
		fmt.Printf("APM Infrastructure Summary:\n")
		fmt.Printf("  Total Stacks: %d\n", summary.TotalStacks)
		fmt.Printf("  Healthy: %d, Degraded: %d, Unhealthy: %d\n", 
			summary.HealthyStacks, summary.DegradedStacks, summary.UnhealthyStacks)
		fmt.Printf("  Last Updated: %s\n", summary.LastUpdated.Format(time.RFC3339))

		fmt.Println("  Resource Summary:")
		fmt.Printf("    Load Balancers: %d\n", summary.ResourceSummary.LoadBalancers)
		fmt.Printf("    ECS Services: %d\n", summary.ResourceSummary.ECSServices)
		fmt.Printf("    RDS Instances: %d\n", summary.ResourceSummary.RDSInstances)
		fmt.Printf("    Lambda Functions: %d\n", summary.ResourceSummary.LambdaFunctions)
		fmt.Printf("    ElastiCache Clusters: %d\n", summary.ResourceSummary.ElastiCacheClusters)
		fmt.Printf("    S3 Buckets: %d\n", summary.ResourceSummary.S3Buckets)
		fmt.Printf("    VPCs: %d\n", summary.ResourceSummary.VPCs)

		if len(summary.RegionSummary) > 0 {
			fmt.Println("  By Region:")
			for region, regionSummary := range summary.RegionSummary {
				fmt.Printf("    %s: %d stacks (%d healthy)\n", 
					region, regionSummary.StackCount, regionSummary.HealthyStacks)
				if len(regionSummary.Issues) > 0 {
					fmt.Printf("      Issues: %d\n", len(regionSummary.Issues))
				}
			}
		}
	}

	// Test 7: Search for specific APM resources
	fmt.Println("\n7. Searching for APM resources...")
	resourceTypes := []string{"loadbalancer", "ecs", "rds", "lambda"}
	
	for _, resourceType := range resourceTypes {
		fmt.Printf("\nSearching for %s resources...\n", resourceType)
		results, err := provider.SearchAPMResources(ctx, resourceType, []string{"us-east-1", "us-west-2"})
		if err != nil {
			log.Printf("Error searching for %s: %v", resourceType, err)
		} else {
			fmt.Printf("Found %d %s resources\n", len(results), resourceType)
			for i, result := range results {
				if i >= 3 { // Show only first 3
					fmt.Printf("  ... and %d more\n", len(results)-3)
					break
				}
				fmt.Printf("  - %s (%s) in %s: %s\n", 
					result.ResourceName, result.ResourceType, result.StackName, result.Status)
				if result.Endpoint != "" {
					fmt.Printf("    Endpoint: %s\n", result.Endpoint)
				}
			}
		}
	}

	fmt.Println("\n✅ CloudFormation stack detection demonstration completed!")
	
	// Export summary to JSON file
	if summary != nil {
		fmt.Println("\n8. Exporting summary to JSON...")
		jsonData, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			log.Printf("Error marshaling summary: %v", err)
		} else {
			err = os.WriteFile("apm-infrastructure-summary.json", jsonData, 0644)
			if err != nil {
				log.Printf("Error writing summary file: %v", err)
			} else {
				fmt.Println("Summary exported to apm-infrastructure-summary.json")
			}
		}
	}
}

func printAPMResources(resources *cloud.APMResources) {
	if len(resources.LoadBalancers) > 0 {
		fmt.Printf("    Load Balancers: %d\n", len(resources.LoadBalancers))
		for _, lb := range resources.LoadBalancers {
			fmt.Printf("      - %s (%s): %s\n", lb.DNSName, lb.Type, lb.Scheme)
		}
	}

	if len(resources.ECSServices) > 0 {
		fmt.Printf("    ECS Services: %d\n", len(resources.ECSServices))
		for _, svc := range resources.ECSServices {
			fmt.Printf("      - %s in %s: %d/%d tasks\n", 
				svc.ServiceName, svc.ClusterName, svc.RunningCount, svc.DesiredCount)
		}
	}

	if len(resources.RDSInstances) > 0 {
		fmt.Printf("    RDS Instances: %d\n", len(resources.RDSInstances))
		for _, db := range resources.RDSInstances {
			fmt.Printf("      - %s (%s): %s\n", 
				db.DBInstanceIdentifier, db.Engine, db.Status)
		}
	}

	if len(resources.LambdaFunctions) > 0 {
		fmt.Printf("    Lambda Functions: %d\n", len(resources.LambdaFunctions))
		for _, fn := range resources.LambdaFunctions {
			fmt.Printf("      - %s (%s): %s\n", 
				fn.FunctionName, fn.Runtime, fn.State)
		}
	}

	if len(resources.ElastiCacheClusters) > 0 {
		fmt.Printf("    ElastiCache Clusters: %d\n", len(resources.ElastiCacheClusters))
		for _, cache := range resources.ElastiCacheClusters {
			fmt.Printf("      - %s (%s): %s\n", 
				cache.ClusterID, cache.Engine, cache.Status)
		}
	}

	if len(resources.S3Buckets) > 0 {
		fmt.Printf("    S3 Buckets: %d\n", len(resources.S3Buckets))
		for _, bucket := range resources.S3Buckets {
			fmt.Printf("      - %s in %s\n", bucket.BucketName, bucket.Region)
		}
	}

	if len(resources.VPCResources) > 0 {
		fmt.Printf("    VPCs: %d\n", len(resources.VPCResources))
		for _, vpc := range resources.VPCResources {
			fmt.Printf("      - %s (%s): %d subnets\n", 
				vpc.VpcId, vpc.CidrBlock, len(vpc.SubnetIds))
		}
	}
}