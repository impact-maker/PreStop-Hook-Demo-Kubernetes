package e2enode
 
import (
    "context"
    "fmt"
    "time"
 
    v1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/kubernetes/test/e2e/framework"
    e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
    imageutils "k8s.io/kubernetes/test/utils/image"
 
    "github.com/onsi/ginkgo/v2"
    "github.com/onsi/gomega"
)
 
// Helper functions
func boolPtr(b bool) *bool {
    return &b
}
 
func int64Ptr(i int64) *int64 {
    return &i
}


var _ = ginkgo.Describe("PreStop Hook TestSuite", func() {

    f := framework.NewDefaultFramework("prestop-hook-testsuite")
    var podClient *e2epod.PodClient
 
    ginkgo.BeforeEach(func() {
        podClient = e2epod.NewPodClient(f)
    })
 
    ginkgo.It("should run PreStop hook basic test case and verify execution", func(ctx context.Context) {
		TestName("PreStop Hook Basic")
	    TestStep("When you have a PreStop hook defined on your container, it will execute before the container is terminated.")
        
		ginkgo.By("Creating a Pod with PreStop hook that writes to a shared volume")
		TestLog("Creating a Pod with PreStop hook that writes to a shared volume")
		
        // Define an emptyDir volume
        volumeName := "shared-data"
        mountPath := "/data"
 
        // Define the lifecycle with the PreStop hook that writes to a file in the volume
        lifecycle := &v1.Lifecycle{
            PreStop: &v1.LifecycleHandler{
                Exec: &v1.ExecAction{
                    Command: []string{
                        "sh",
                        "-c",
                        fmt.Sprintf("echo 'PreStop Hook Executed' > %s/prestop_hook_executed; sleep 10", mountPath),
                    },
                },
            },
        }
        TestStep("Imagine You Have a Pod with the Following Specification")
        // Define the pod with the emptyDir volume and appropriate security context
        podWithHook := &v1.Pod{
            ObjectMeta: metav1.ObjectMeta{
                Name: "prestop-pod",
            },
            Spec: v1.PodSpec{
                Containers: []v1.Container{
                    {
                        Name:    "prestop-container",
                        Image:   imageutils.GetE2EImage(imageutils.Agnhost),
                        Args:    []string{"pause"},
                        Lifecycle: lifecycle,
                        VolumeMounts: []v1.VolumeMount{
                            {
                                Name:      volumeName,
                                MountPath: mountPath,
                            },
                        },
                        SecurityContext: &v1.SecurityContext{
                            RunAsNonRoot:             boolPtr(true),
                            RunAsUser:                int64Ptr(1000),
                            RunAsGroup:               int64Ptr(1000),
                            AllowPrivilegeEscalation: boolPtr(false),
                            Capabilities: &v1.Capabilities{
                                Drop: []v1.Capability{"ALL"},
                            },
                            SeccompProfile: &v1.SeccompProfile{
                                Type: v1.SeccompProfileTypeRuntimeDefault,
                            },
                        },
                    },
                },
                Volumes: []v1.Volume{
                    {
                        Name: volumeName,
                        VolumeSource: v1.VolumeSource{
                            EmptyDir: &v1.EmptyDirVolumeSource{
                                Medium: v1.StorageMediumDefault,
                            },
                        },
                    },
                },
                // Increase termination grace period to ensure the pod remains accessible
                TerminationGracePeriodSeconds: int64Ptr(60),
            },
        }
	
		PodSpec(podWithHook)

        // Create and run the pod
        TestStep("This Pod should start successfully and run for some time")
        createdPod := podClient.CreateSync(ctx, podWithHook)
		ginkgo.By("Pod is created successfully")
		TestLog("Pod is created successfully")
        
        // Wait for the pod to be running
        err := e2epod.WaitForPodRunningInNamespace(ctx, f.ClientSet, createdPod)
        framework.ExpectNoError(err)
 
        TestLog("The Pod is running successfully")

        // Delete the pod to trigger the PreStop hook
        TestStep("When the container is terminated, the PreStop hook will be triggered within the grace period")
        err = podClient.Delete(ctx, createdPod.Name, metav1.DeleteOptions{})
        framework.ExpectNoError(err)
        TestLog("Pod was deleted successfully")
        

        // Check if the file exists inside the container
        TestStep("Once the PreStop hook is successfully executed, the container terminates successfully")
        cmd := []string{"cat", fmt.Sprintf("%s/prestop_hook_executed", mountPath)}
        stdout, stderr, err := e2epod.ExecWithOptions(f, e2epod.ExecOptions{
            Command:       cmd,
            Namespace:     createdPod.Namespace,
            PodName:       createdPod.Name,
            ContainerName: "prestop-container",
            CaptureStdout: true,
            CaptureStderr: true,
        })
		ginkgo.By("Pod Events and debug")
		TestLog("Pod Events and debug")
        if err != nil {
            // Print pod events for debugging
			
            events, listErr := f.ClientSet.CoreV1().Events(createdPod.Namespace).List(ctx, metav1.ListOptions{
                FieldSelector: fmt.Sprintf("involvedObject.name=%s", createdPod.Name),
            })
            if listErr == nil {
                TestStep("Pod Events:")
                for _, event := range events.Items {
                    TestLog("Event Reason:")
                    TestLog(event.Reason) 
                    TestLog("Message:")
                    TestLog(event.Message)
                }
            }
            framework.Failf("PreStop hook did not execute successfully or file not found: %v. Stderr: %s", err, stderr)
        } else {
            TestStep("PreStop hook logs will be created")
            TestLog(stdout)
            gomega.Expect(stdout).To(gomega.ContainSubstring("PreStop Hook Executed"))
            TestLog("PreStop hook executed successfully")
        }
 
        // Wait for the pod to be deleted
        err = e2epod.WaitForPodNotFoundInNamespace(ctx, f.ClientSet, createdPod.Name, createdPod.Namespace, 2*time.Minute)
        framework.ExpectNoError(err)
        TestLog("Waiting for the Pod to be deleted")
    })
})
