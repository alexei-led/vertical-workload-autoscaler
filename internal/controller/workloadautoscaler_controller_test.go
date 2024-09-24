package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vwav1 "github.com/alexei-led/vertical-workload-autoscaler/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("VerticalWorkloadAutoscaler Controller", func() {
	const resourceName = "test-resource"
	const namespace = "default"

	var (
		ctx                        context.Context
		typeNamespacedName         types.NamespacedName
		verticalWorkloadAutoscaler *vwav1.VerticalWorkloadAutoscaler
		controllerReconciler       *VerticalWorkloadAutoscalerReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}
		verticalWorkloadAutoscaler = &vwav1.VerticalWorkloadAutoscaler{}
		controllerReconciler = &VerticalWorkloadAutoscalerReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		By("creating the custom resource for the Kind VerticalWorkloadAutoscaler")
		err := k8sClient.Get(ctx, typeNamespacedName, verticalWorkloadAutoscaler)
		if err != nil && errors.IsNotFound(err) {
			resource := &vwav1.VerticalWorkloadAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
				// TODO: Specify other spec details if needed.
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		}
	})

	AfterEach(func() {
		By("cleaning up the custom resource for the Kind VerticalWorkloadAutoscaler")
		resource := &vwav1.VerticalWorkloadAutoscaler{}
		err := k8sClient.Get(ctx, typeNamespacedName, resource)
		if err == nil {
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		}
	})

	It("should successfully reconcile the resource", func() {
		By("Reconciling the created resource")
		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: typeNamespacedName,
		})
		Expect(err).NotTo(HaveOccurred())

		// Example: If you expect a certain status condition after reconciliation, verify it here.
		Eventually(func() error {
			return k8sClient.Get(ctx, typeNamespacedName, verticalWorkloadAutoscaler)
		}).Should(Succeed())
	})

	It("should delay update if outside allowed update window", func() {
		wa := &vwav1.VerticalWorkloadAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: vwav1.VerticalWorkloadAutoscalerSpec{
				AllowedUpdateWindows: []vwav1.UpdateWindow{
					{
						DayOfWeek: "Monday",
						StartTime: "10:00",
						EndTime:   "12:00",
						TimeZone:  "UTC",
					},
				},
			},
		}

		// Create the VerticalWorkloadAutoscaler object
		Expect(k8sClient.Create(ctx, wa)).Should(Succeed())
		defer func() { Expect(k8sClient.Delete(ctx, wa)).To(Succeed()) }()

		// Mock current time to be outside the allowed update window
		now := time.Date(2024, time.September, 21, 9, 0, 0, 0, time.UTC)
		originalTimeNow := timeNow
		timeNow = func() time.Time { return now }
		defer func() { timeNow = originalTimeNow }()

		// Reconcile
		req := reconcile.Request{
			NamespacedName: typeNamespacedName,
		}
		_, err := controllerReconciler.Reconcile(ctx, req)
		Expect(err).ToNot(HaveOccurred())

		// Check if the update was delayed
		Eventually(func() error {
			return k8sClient.Get(ctx, req.NamespacedName, wa)
		}).Should(Succeed())
		Expect(wa.Status.LastUpdated).To(BeNil())
	})
})
