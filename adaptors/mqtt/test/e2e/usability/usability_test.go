package usability_test

import (
	"encoding/base64"
	"fmt"
	"path/filepath"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rancher/octopus/adaptors/mqtt/api/v1alpha1"
	edgev1alpha1 "github.com/rancher/octopus/api/v1alpha1"
	"github.com/rancher/octopus/test/util/exec"
	"github.com/rancher/octopus/test/util/node"
)

var (
	testDeviceLink          edgev1alpha1.DeviceLink
	testDeviceLinkName      = "kitchen-light"
	testDeviceLinkNamespace = "default"
	subscribedMessage       string
	temporaryMessage        string
)

var _ = Describe("verify usability", func() {

	BeforeEach(func() {
		deployMQTTDeviceLink()
	})

	AfterEach(func() {
		_ = k8sCli.DeleteAllOf(testCtx, &edgev1alpha1.DeviceLink{}, client.InNamespace(testDeviceLink.Namespace))
	})

	Context("modify MQTT device link spec", func() {

		Specify("if invalid node spec", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when invalid node spec", invalidNodeSpec)

			By("then node of the device link is not found", isNodeExistedFalse)

			By("when correct node spec", correctNodeSpec)

			By("then node of the device link is found", isNodeExistedTrue)

		})

		Specify("if invalid model spec", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when invalid model spec", invalidModelSpec)

			By("then model of the device link is not found", isModelExistedFalse)

			By("when correct model spec", correctModelSpec)

			By("then model of the device link is found", isModelExistedTrue)

		})

		Specify("if invalid adaptor spec", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when invalid adaptor spec", invalidAdaptorSpec)

			By("then adaptor of the device link is not found", isAdaptorExistedFalse)

			By("when correct adaptor spec", correctAdaptorSpec)

			By("then adaptor of the device link is found", isAdaptorExistedTrue)

		})

		Specify("if invalid device spec", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when invalid device spec", invalidDeviceSpec)

			By("then the device link is not created", isDeviceConnectedFalse)

			By("when correct device spec", correctDeviceSpec)

			By("then the device link is connected", isDeviceConnectedTrue)

		})
	})

	Context("publish/subscribe messages through MQTT device link", func() {

		// unable to publish message through simulator
		PSpecify("publish messages through an MQTT device link", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when publish a message to corresponding MQTT server", publishMessage)

			By("then the message should be inside device link", isPublishedMessageInsideDeviceLink)

		})

		PSpecify("subscribe messages through an MQTT device link", func() {

			// By("when deploy an MQTT device link with remote server", deployDeviceLinkWithRemoteServer)

			By("then the device link is connected", isDeviceConnectedTrue)

			By("when subscribe a message from server", subscribeMessage)

			By("then the message is same to the value in device link", isSubscribedMessageValid)

		})

		PSpecify("subscribe messages through an MQTT device link which does not exist", func() {

			// By("when deploy MQTT device link with remote server", deployDeviceLinkWithRemoteServer)

			By("then the device link is connected", isDeviceConnectedTrue)

			By("when subscribe a message and save it", subscribeMessageAndSave)

			By("the the message is subscribed", isSubscribedMessageValid)

			By("when delete MQTT device link", deleteMQTTDeviceLink)

			By("then the device link is deleted", isDeviceLinkDeleted)

			By("when subscribe a message again", subscribeMessage)

			By("then the two messages are the same", isMessagesTheSame)

		})

		PSpecify("subscribe messages through an MQTT device link whose payload is null", func() {

			By("when deploy an MQTT device link with payload null", deployDeviceLinkWithPayloadNull)

			By("then the device link is connected", isDeviceConnectedTrue)

			By("when subscribe a message", subscribeMessage)

			By("then the message is null", isSubscribeMessageNull)

		})

		Specify("if the MQTT server is unavailable", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when invalid server URL", invalidServerURL)

			By("then the device link is not connected", isDeviceCreatedFalse)

			By("when correct server URL", correctServerURL)

			By("then the device link is connected", isDeviceConnectedTrue)

		})

		PSpecify("subscribe messages through an MQTT device link whose payload is a complex JSON", func() {

			By("when deploy an MQTT device link whose payload is a complex JSON", deployDeviceLinkWithComplexJSONPayload)

			By("then the device link is connected", isDeviceConnectedTrue)

			By("when subscribe a message", subscribeMessage)

			By("then the subscribed message is same to the JSON payload", isSubscribedMessageValid)

		})
	})

	Context("operation on pods/CRD/nodes", func() {

		Specify("if delete adaptor pods", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when delete MQTT adaptor pods", deleteMQTTAdaptorPods)

			By("then adaptor of the device link is not found", isAdaptorExistedFalse)

		})

		Specify("if delete limbs pods", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when delete limbs pods", deleteLimbsPods)

			By("then the MQTT adaptor pods become error", isMQTTAdaptorPodsError)

		})

		Specify("if delete MQTT device model", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when delete MQTT device model", deleteMQTTDeviceModel)

			By("then model of the device link is not found", isModelExistedFalse)

			By("redeploy MQTT device model", redeployMQTTDeviceModel)

		})

		Specify("if delete corresponding node", func() {

			By("given the device link is connected", isDeviceConnectedTrue)

			By("when delete corresponding cluster node", deleteCorrespondingNode)

			By("then node of the device link is not found", isNodeExistedFalse)

		})
	})
})

type judgeFunc func(edgev1alpha1.DeviceLink) bool

func doDeviceLinkJudgment(judge judgeFunc) {
	Eventually(func() bool {
		var deviceLinkKey = types.NamespacedName{
			Name:      testDeviceLinkName,
			Namespace: testDeviceLinkNamespace,
		}
		if err := k8sCli.Get(testCtx, deviceLinkKey, &testDeviceLink); err != nil {
			Fail(err.Error())
		}
		return judge(testDeviceLink)
	}, 300, 3).Should(BeTrue())
}

func getDeviceLinkPointer() *edgev1alpha1.DeviceLink {
	var deviceLinkKey = types.NamespacedName{
		Namespace: testDeviceLink.Namespace,
		Name:      testDeviceLink.Name,
	}
	if err := k8sCli.Get(testCtx, deviceLinkKey, &testDeviceLink); err != nil {
		Fail(err.Error())
	}
	return &testDeviceLink
}

func deployMQTTDeviceLink() {
	Expect(exec.RunKubectl(nil, GinkgoWriter, "apply", "-f", filepath.Join(testCurrDir, "deploy", "e2e", "dl_attributed_topic_kitchen_light.yaml"))).
		Should(Succeed())
}

func deployDeviceLinkWithPayloadNull() {
	Expect(exec.RunKubectl(nil, GinkgoWriter, "apply", "-f", filepath.Join(testCurrDir, "test", "e2e", "usability", "testdata", "dl_attributed_topic_payload_null.yaml"))).
		Should(Succeed())
}

func deployDeviceLinkWithComplexJSONPayload() {
	Expect(exec.RunKubectl(nil, GinkgoWriter, "apply", "-f", filepath.Join(testCurrDir, "test", "e2e", "usability", "testdata", "dl_attributed_topic_complex_json.yaml"))).
		Should(Succeed())
}

func correctNodeSpec() {
	var targetNode, err = node.GetValidWorker(testCtx, k8sCli)
	Expect(err).ShouldNot(HaveOccurred())
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(fmt.Sprintf(`{"spec":{"adaptor":{"node":"%s"}}}`, targetNode))
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func invalidNodeSpec() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"adaptor":{"node":"wrong-node"}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func isNodeExistedTrue() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetNodeExistedStatus() == metav1.ConditionTrue
	}
	doDeviceLinkJudgment(judge)
}

func isNodeExistedFalse() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetNodeExistedStatus() == metav1.ConditionFalse
	}
	doDeviceLinkJudgment(judge)
}

func correctModelSpec() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"model":{"apiVersion":"devices.edge.cattle.io/v1alpha1"}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func invalidModelSpec() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"model":{"apiVersion":"wrong-apiVersion"}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func isModelExistedTrue() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetModelExistedStatus() == metav1.ConditionTrue
	}
	doDeviceLinkJudgment(judge)
}

func isModelExistedFalse() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetModelExistedStatus() == metav1.ConditionFalse
	}
	doDeviceLinkJudgment(judge)
}

func correctAdaptorSpec() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"adaptor":{"name":"adaptors.edge.cattle.io/mqtt"}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func invalidAdaptorSpec() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"adaptor":{"name":"wrong-adaptor-name"}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func isAdaptorExistedTrue() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetAdaptorExistedStatus() == metav1.ConditionTrue
	}
	doDeviceLinkJudgment(judge)
}

func isAdaptorExistedFalse() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetAdaptorExistedStatus() == metav1.ConditionFalse
	}
	doDeviceLinkJudgment(judge)
}

func correctDeviceSpec() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"template":{"spec":{"protocol":{"pattern":"AttributedTopic"}}}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func invalidDeviceSpec() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"template":{"spec":{"protocol":{"pattern":"wrong-pattern"}}}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func isDeviceConnectedTrue() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetDeviceConnectedStatus() == metav1.ConditionTrue
	}
	doDeviceLinkJudgment(judge)
}

func isDeviceConnectedFalse() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetDeviceConnectedStatus() == metav1.ConditionFalse
	}
	doDeviceLinkJudgment(judge)
}

func isDeviceCreatedFalse() {
	var judge = func(deviceLink edgev1alpha1.DeviceLink) bool {
		return deviceLink.GetDeviceCreatedStatus() == metav1.ConditionFalse || deviceLink.GetDeviceConnectedStatus() == metav1.ConditionFalse
	}
	doDeviceLinkJudgment(judge)
}

func deleteMQTTAdaptorPods() {
	Expect(k8sCli.DeleteAllOf(testCtx, &corev1.Pod{}, client.InNamespace("octopus-system"), client.MatchingLabels{"app.kubernetes.io/name": "octopus-adaptor-mqtt"})).
		Should(Succeed())
}

func deleteLimbsPods() {
	Expect(k8sCli.DeleteAllOf(testCtx, &corev1.Pod{}, client.InNamespace("octopus-system"), client.MatchingLabels{"app.kubernetes.io/component": "limb"})).
		Should(Succeed())
}

func isMQTTAdaptorPodsError() {
	var podList corev1.PodList
	Eventually(func() bool {
		if err := k8sCli.List(testCtx, &podList, client.InNamespace("octopus-system"), client.MatchingLabels{"app.kubernetes.io/name": "octopus-adaptor-mqtt"}); err != nil {
			Fail(err.Error())
		}
		for _, pod := range podList.Items {
			for _, condition := range pod.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == "False" {
					return true
				}
			}
		}
		return false
	}, 300, 1).Should(BeTrue())
}

func deleteCorrespondingNode() {
	var targetNode, err = node.GetValidWorker(testCtx, k8sCli)
	Expect(err).ShouldNot(HaveOccurred())

	var correspondingNode = corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: targetNode,
		},
	}
	Expect(k8sCli.Delete(testCtx, &correspondingNode)).Should(Succeed())
}

func deleteMQTTDeviceModel() {
	var crd = v1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mqttdevices.devices.edge.cattle.io",
		},
	}
	Expect(k8sCli.Delete(testCtx, &crd)).Should(Succeed())
}

func redeployMQTTDeviceModel() {
	Expect(exec.RunKubectl(nil, GinkgoWriter, "apply", "-f", filepath.Join(testCurrDir, "deploy", "manifests", "crd", "base", "devices.edge.cattle.io_mqttdevices.yaml"))).
		Should(Succeed())
}

func invalidServerURL() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"template":{"spec":{"protocol":{"client":{"server":"wrong-server"}}}}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func correctServerURL() {
	deviceLinkPtr := getDeviceLinkPointer()
	patch := []byte(`{"spec":{"template":{"spec":{"protocol":{"client":{"server":"tcp://octopus-simulator-mqtt.octopus-simulator-system:1883"}}}}}}`)
	Expect(k8sCli.Patch(testCtx, deviceLinkPtr, client.RawPatch(types.MergePatchType, patch))).Should(Succeed())
}

func publishMessage() {
	var podList corev1.PodList
	Expect(k8sCli.List(testCtx, &podList, client.InNamespace("default"))).Should(Succeed())
	for _, pod := range podList.Items {
		Expect(exec.RunKubectl(nil, GinkgoWriter, "exec", pod.Name, "--", "mosquitto_pub", "-h", "octopus-simulator-mqtt.octopus-simulator-system",
			"-t", "cattle.io/octopus/home/control/kitchen/light/gear", "-m", "high")).Should(Succeed())
	}
}

func isPublishedMessageInsideDeviceLink() {
	var device v1alpha1.MQTTDevice
	count := 0
	Eventually(func() bool {
		count++
		var targetKey = types.NamespacedName{
			Namespace: testDeviceLinkNamespace,
			Name:      testDeviceLinkName,
		}
		if err := k8sCli.Get(testCtx, targetKey, &device); err != nil {
			Fail(err.Error())
		}
		if len(device.Status.Properties) > 0 {
			for _, property := range device.Status.Properties {
				if property.Name == "gear" {
					return string(property.Value.Raw) == "\""+base64.StdEncoding.EncodeToString([]byte("high"))+"\""
				}
			}
		}
		if count%10 == 0 && len(device.Status.Properties) == 0 {
			publishMessage()
		}
		return false
	}, 300, 1).Should(BeTrue())
}

func subscribeMessage() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://test.mosquitto.org:1883")

	choke := make(chan [2]string)

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		choke <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	testClient := mqtt.NewClient(opts)
	if token := testClient.Connect(); token.Wait() && token.Error() != nil {
		Fail(token.Error().Error())
	}

	topic := "cattle.io/octopus/home/status/kitchen/light/gear"
	if token := testClient.Subscribe(topic, byte(1), nil); token.Wait() && token.Error() != nil {
		Fail(token.Error().Error())
	}

	incoming := <-choke
	subscribedMessage = incoming[1]

	testClient.Disconnect(250)
}

func isSubscribedMessageValid() {
	var targetKey = types.NamespacedName{
		Namespace: testDeviceLink.Namespace,
		Name:      testDeviceLink.Name,
	}
	var device v1alpha1.MQTTDevice
	Eventually(func() bool {
		if err := k8sCli.Get(testCtx, targetKey, &device); err != nil {
			Fail(err.Error())
		}
		for _, property := range device.Spec.Properties {
			if property.Name == "gear" {
				return string(property.Value.Raw) == subscribedMessage
			}
		}
		return false
	}, 300, 1).Should(BeTrue())
}

func subscribeMessageAndSave() {
	subscribeMessage()
	temporaryMessage = subscribedMessage
}

func isMessagesTheSame() {
	Expect(temporaryMessage == subscribedMessage).Should(BeTrue())
}

func isSubscribeMessageNull() {
	Expect(subscribedMessage == "null").Should(BeTrue())
}

func deleteMQTTDeviceLink() {
	var targetKey = types.NamespacedName{
		Namespace: testDeviceLink.Namespace,
		Name:      testDeviceLink.Name,
	}
	if err := k8sCli.Get(testCtx, targetKey, &testDeviceLink); err != nil {
		Fail(err.Error())
	}
	Expect(k8sCli.Delete(testCtx, &testDeviceLink)).Should(Succeed())
}

func isDeviceLinkDeleted() {
	Eventually(func() bool {
		var deviceLinkKey = types.NamespacedName{
			Name:      testDeviceLink.Name,
			Namespace: testDeviceLink.Namespace,
		}
		if err := k8sCli.Get(testCtx, deviceLinkKey, &testDeviceLink); err != nil {
			return true
		}
		return false
	}, 300, 3).Should(BeTrue())
}
