package test

import (
	"testing"
	"fmt"
	 "strings"
	 "time"
	
	"github.com/gruntwork-io/terratest/modules/azure"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/gruntwork-io/terratest/modules/ssh"

)

// You normally want to run this under a separate "Testing" subscription
// For lab purposes you will use your assigned subscription under the Cloud Dev/Ops program tenant
//var subscriptionID string = "<your-azure-subscription-id"
var subscriptionID string = "a14d25e0-b8df-4651-933c-cb1ce9a3ba5a"

func TestAzureLinuxVMCreation(t *testing.T) {
	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../",
		// Override the default terraform variables
		Vars: map[string]interface{}{
			"labelPrefix": "tran0507",
		},
	}

	// Run `terraform init` and `terraform apply`. Fail the test if there are any errors.
	terraform.InitAndApply(t, terraformOptions)
    defer func() {
        if !t.Failed() {
            terraform.Destroy(t, terraformOptions)
        }
    }()

	// Run `terraform output` to get the value of output variable
	vmName := terraform.Output(t, terraformOptions, "vm_name")
	//vmName:="tran0507A05VM"
	resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")
	nicName := terraform.Output(t, terraformOptions, "nic_name") // Fetch NIC name
	publicIP := terraform.Output(t, terraformOptions, "public_ip") // Fetch Public IP

	// Confirm VM exists
	assert.True(t, azure.VirtualMachineExists(t, vmName, resourceGroupName, subscriptionID))
	time.Sleep(30* time.Second)

	vm:= azure.GetVirtualMachine(t,vmName,resourceGroupName,subscriptionID)
	
	// Get NIC attached to the VM
	vmNICs := *vm.NetworkProfile.NetworkInterfaces
	assert.NotEmpty(t, vmNICs, "VM should have at least one NIC")

	// Check if the NIC attached to the VM matches the one created by Terraform
	found := false
	for _, nic := range vmNICs {
		if nic.ID != nil {
			nicNameFromID, err := extractResourceNameFromID(*nic.ID)
			assert.NoError(t, err, "Failed to extract NIC name from ID")
			if nicNameFromID == nicName {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "The NIC created by Terraform should be attached to the VM")

	
	// Check if the public IP is valid
	assert.NotEmpty(t, publicIP, "Public IP should not be empty")

	// SSH into the VM and check Ubuntu version

	keyPair := &ssh.KeyPair{
		PrivateKey: "/mnt/c/Users/X/.ssh/id_rsa",
		PublicKey:  "/mnt/c/Users/X/.ssh/id_rsa.pub",
	}
	
	sshHost := ssh.Host{
		Hostname:    terraform.Output(t, terraformOptions, "public_ip"),
		SshUserName: terraform.Output(t, terraformOptions, "admin_username"),
		SshKeyPair:  keyPair,
	}

	// Command to check Ubuntu version
	command := "cat /etc/os-release && lsb_release -a || true"

	output, err := ssh.CheckSshCommandE(t, sshHost, command)
	if err != nil {
		t.Logf("SSH command output: %s", output)
		t.Fatalf("Failed to execute command: %v", err)
	}
	t.Logf("OS release info: %s", output)
	assert.Contains(t, output, "Ubuntu 22.04", "VM should be running Ubuntu 22.04")


}

// Helper function to extract resource name from Azure resource ID
func extractResourceNameFromID(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1], nil
	}
	return "", fmt.Errorf("invalid resource ID format")
}