package adaptor

import (
	"io"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rancher/octopus/adaptors/ble/pkg/adaptor"
	"github.com/rancher/octopus/pkg/adaptor/api/v1alpha1"
	mock_v1alpha1 "github.com/rancher/octopus/pkg/adaptor/api/v1alpha1/mock"
)

// testing scenarios:
// 	+ Server
//		- validate if the connection stop when it closes
//		- validate the process of input model
//      - validate the recognition of the input model
// 		- validate the process of input device
var _ = Describe("Connection", func() {
	var (
		err error

		mockCtrl *gomock.Controller
		service  *adaptor.Service
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		service = adaptor.NewService()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Server", func() {

		var mockServer *mock_v1alpha1.MockConnection_ConnectServer

		BeforeEach(func() {
			mockServer = mock_v1alpha1.NewMockConnection_ConnectServer(mockCtrl)
		})

		It("should be stopped if closed", func() {
			// io.EOF
			mockServer.EXPECT().Recv().Return(nil, io.EOF)
			err = service.Connect(mockServer)
			Expect(err).ToNot(HaveOccurred())

			// canceled by context
			mockServer.EXPECT().Recv().Return(nil, status.Error(grpccodes.Canceled, "context canceled"))
			err = service.Connect(mockServer)
			Expect(err).ToNot(HaveOccurred())

			// other canceled reason
			mockServer.EXPECT().Recv().Return(nil, status.Error(grpccodes.Canceled, "other"))
			err = service.Connect(mockServer)
			Expect(err).To(HaveOccurred())

			// transport is closing
			mockServer.EXPECT().Recv().Return(nil, status.Error(grpccodes.Unavailable, "transport is closing"))
			err = service.Connect(mockServer)
			Expect(err).ToNot(HaveOccurred())

			// other unavailable reason
			mockServer.EXPECT().Recv().Return(nil, status.Error(grpccodes.Unavailable, "other"))
			err = service.Connect(mockServer)
			Expect(err).To(HaveOccurred())
		})

		It("should process the input device", func() {
			// failed unmarshal
			mockServer.EXPECT().Recv().Return(&v1alpha1.ConnectRequest{
				Model: &metav1.TypeMeta{
					APIVersion: "devices.edge.cattle.io/v1alpha1",
					Kind:       "BluetoothDevice",
				},
				Device: []byte(`{this is an illegal json}`),
			}, nil)
			err = service.Connect(mockServer)
			var sts = status.Convert(err)
			Expect(sts.Code()).To(Equal(grpccodes.InvalidArgument))
			Expect(sts.Message()).To(HavePrefix("failed to unmarshal device"))

			// correct logic
			mockServer.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
			mockServer.EXPECT().Recv().Return(&v1alpha1.ConnectRequest{
				Model: &metav1.TypeMeta{
					APIVersion: "devices.edge.cattle.io/v1alpha1",
					Kind:       "BluetoothDevice",
				},
				Device: []byte(`
				{
					"apiVersion":"edge.cattle.io/v1alpha1",
					"kind":"DeviceLink",
					"metadata":{
						"name":"xiaomi-temp-rs2200",
						"namespace":"default"
					},
					"spec":{
						"template":{
							"spec":{
								"protocol":{
									"name":"MJ_HT_V1",
									"macAddress":""
								},
								"properties":[
									{
										"name":"data",
										"description":"XiaoMi temp sensor with temperature and humidity data",
										"accessMode":"NotifyOnly",
										"visitor":{
											"characteristicUUID":"226c000064764566756266734470666d"
										}
									}
								]
							}
						}
					}
				}`),
			}, nil)
			mockServer.EXPECT().Recv().Return(nil, io.EOF) // simulate to close the connection
			err = service.Connect(mockServer)
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
