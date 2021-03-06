package mictest

import (
	"testing"
	"time"

	aadpodid "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	cp "github.com/Azure/aad-pod-identity/pkg/cloudprovider/cloudprovidertest"
	crd "github.com/Azure/aad-pod-identity/pkg/crd/crdtest"
	pod "github.com/Azure/aad-pod-identity/pkg/pod/podtest"
	"github.com/golang/glog"
)

func TestMapMICClient(t *testing.T) {
	micClient := &TestMICClient{}

	idList := make([]aadpodid.AzureIdentity, 0)

	id := new(aadpodid.AzureIdentity)
	id.Name = "test-azure-identity"

	idList = append(idList, *id)

	id.Name = "test-akssvcrg-id"
	idList = append(idList, *id)

	idMap, _ := micClient.ConvertIDListToMap(&idList)

	name := "test-azure-identity"
	count := 3
	if azureID, idPresent := idMap[name]; idPresent {
		if azureID.Name != name {
			panic("id map id value mismatch")
		}
		count = count - 1
	}

	name = "test-akssvcrg-id"
	if azureID, idPresent := idMap[name]; idPresent {
		if azureID.Name != name {
			panic("id map id value mismatch")
		}
		count = count - 1
	}

	name = "test not there"
	if _, idPresent := idMap[name]; idPresent {
		panic("not present found")
	} else {
		count = count - 1
	}
	if count != 0 {
		panic("Test count mismatch")
	}

}

func TestSimpleMICClient(t *testing.T) {

	exit := make(<-chan struct{}, 0)
	eventCh := make(chan aadpodid.EventType, 100)
	cloudClient := cp.NewTestCloudClient()
	crdClient := crd.NewTestCrdClient(nil)
	podClient := pod.NewTestPodClient()

	micClient := NewMICClient(eventCh, cloudClient, crdClient, podClient)

	crdClient.CreateId("test-id", aadpodid.UserAssignedMSI, "test-user-msi-resourceid", "test-user-msi-clientid", nil, "", "", "")
	crdClient.CreateBinding("testbinding", "test-id", "test-select")

	podClient.AddPod("test-pod", "default", "test-node", "test-select")
	podClient.GetPods()

	eventCh <- aadpodid.PodCreated
	go micClient.Sync(exit)
	time.Sleep(2 * time.Second)
	testPass := false
	listAssignedIDs, err := crdClient.ListAssignedIDs()
	if err != nil {
		glog.Error(err)
		panic("list assigned failed")
	}
	if listAssignedIDs != nil {
		for _, assignedID := range *listAssignedIDs {
			if assignedID.Spec.Pod == "test-pod" && assignedID.Spec.PodNamespace == "default" && assignedID.Spec.NodeName == "test-node" &&
				assignedID.Spec.AzureBindingRef.Name == "testbinding" && assignedID.Spec.AzureIdentityRef.Name == "test-id" {
				testPass = true
				break
			}
		}

	}
	if !testPass {
		panic("assigned id mismatch")
	}
}
