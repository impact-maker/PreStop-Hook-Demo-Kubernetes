package e2enode
 
import (
    "context"
    //"time"
 
    "github.com/onsi/ginkgo/v2"
    v1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    admissionapi "k8s.io/pod-security-admission/api"
    "k8s.io/kubernetes/test/e2e/framework"
    e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
   // imageutils "k8s.io/kubernetes/test/utils/image"
)
 
var _ = ginkgo.Describe("PreStop Hook Test Final", func() {
    f := framework.NewDefaultFramework("prestop-hook-test")
    f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged

    ginkgo.It("should call the container's preStop hook and terminate it if its startup probe fails", func() {
    regular1 := "regular-1"

    ginkgo.By("Defining the pod specification with PreStop hook and StartupProbe")
    podSpec := &v1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name: "test-pod",
        },
        Spec: v1.PodSpec{
            RestartPolicy: v1.RestartPolicyNever,
            Containers: []v1.Container{
                {
                    Name:  regular1,
                    Image: busyboxImage,
                    Command: ExecCommand(regular1, execCommand{
                        Delay:              100,
                        TerminationSeconds: 15,
                        ExitCode:           0,
                    }),
                    StartupProbe: &v1.Probe{
                        ProbeHandler: v1.ProbeHandler{
                            Exec: &v1.ExecAction{
                                Command: []string{
                                    "sh",
                                    "-c",
                                    "exit 1",
                                },
                            },
                        },
                        InitialDelaySeconds: 10,
                        FailureThreshold:    1,
                    },
                    Lifecycle: &v1.Lifecycle{
                        PreStop: &v1.LifecycleHandler{
                            Exec: &v1.ExecAction{
                                Command: ExecCommand(prefixedName(PreStopPrefix, regular1), execCommand{
                                    Delay:         1,
                                    ExitCode:      0,
                                    ContainerName: regular1,
                                }),
                            },
                        },
                    },
                },
            },
        },
    }

    ginkgo.By("Preparing the pod")
    preparePod(podSpec)

    client := e2epod.NewPodClient(f)
    ginkgo.By("Creating the pod")
    podSpec = client.Create(context.TODO(), podSpec)

    ginkgo.By("Waiting for the pod to complete")
    err := e2epod.WaitForPodNoLongerRunningInNamespace(context.TODO(), f.ClientSet, podSpec.Name, podSpec.Namespace)
    framework.ExpectNoError(err)

    ginkgo.By("Retrieving the pod specification after completion")
    podSpec, err = client.Get(context.TODO(), podSpec.Name, metav1.GetOptions{})
    framework.ExpectNoError(err)

    ginkgo.By("Parsing the test results")
    results := parseOutput(context.TODO(), f, podSpec)

    ginkgo.By("Analyzing the test results")
    framework.ExpectNoError(results.RunTogether(regular1, prefixedName(PreStopPrefix, regular1)))
    framework.ExpectNoError(results.Starts(prefixedName(PreStopPrefix, regular1)))
    framework.ExpectNoError(results.Exits(regular1))

    ginkgo.By("Test completed successfully")
})

})
