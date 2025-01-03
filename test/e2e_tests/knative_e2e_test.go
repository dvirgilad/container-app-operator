package e2e_tests

import (
	"fmt"

	"k8s.io/client-go/util/retry"

	"github.com/dana-team/container-app-operator/test/e2e_tests/testconsts"

	"github.com/dana-team/container-app-operator/test/e2e_tests/mocks"
	utilst "github.com/dana-team/container-app-operator/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// verifyLatestReadyRevision ensures the readiness of the latest Knative Revision
// if shouldRevisionBeReady is true. It also checks and asserts the state of the LatestReadyRevision.
func verifyLatestReadyRevision(shouldRevisionBeReady bool, name, namespace, latestReadyRevisionBeforeUpdate string) {
	nextRevisionName := utilst.GetNextRevisionName(latestReadyRevisionBeforeUpdate)

	if shouldRevisionBeReady {
		checkRevisionReadiness(nextRevisionName, true)
		By("Ensuring that the capp LatestReadyRevision is updated")
		Eventually(func() string {
			return utilst.GetCapp(k8sClient, name, namespace).Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(nextRevisionName))
	} else {
		By("Ensuring that the capp LatestReadyRevision is not updated")
		checkRevisionReadiness(nextRevisionName, false)
		Eventually(func() string {
			return utilst.GetCapp(k8sClient, name, namespace).Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(latestReadyRevisionBeforeUpdate))
	}
}

// checkRevisionReadiness checks the readiness of the specified revision and asserts its state based on shouldBeReady flag.
func checkRevisionReadiness(revisionName string, shouldBeReady bool) {
	By("Checking if the revision was created successfully")
	revisionObject := mocks.CreateRevisionObject(revisionName)
	Eventually(func() bool {
		return utilst.DoesResourceExist(k8sClient, revisionObject)
	}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "Should find a resource.")
	if shouldBeReady {
		By("Ensuring that the new revision is ready")
		Eventually(func() bool {
			return utilst.GetRevision(k8sClient, revisionObject.Name, revisionObject.Namespace).IsReady()
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())
	} else {
		By("Ensuring that the new revision is not ready")
		Eventually(func() bool {
			return utilst.GetRevision(k8sClient, revisionObject.Name, revisionObject.Namespace).IsReady()
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(BeTrue())
	}
}

// testMetricAnnotation tests capp instance creation with a specified metric annotation.
func testMetricAnnotation(metricType string) {
	By("Creating a capp instance")
	testCapp := mocks.CreateBaseCapp()
	testCapp.Spec.ScaleMetric = metricType
	createdCapp := utilst.CreateCapp(k8sClient, testCapp)

	By(fmt.Sprintf("Checking if the ksvc was created with %s metric annotation successfully", metricType))
	Eventually(func() string {
		ksvc := utilst.GetKSVC(k8sClient, createdCapp.Name, createdCapp.Namespace)
		return ksvc.Spec.Template.Annotations[testconsts.KnativeMetricAnnotation]
	}, testconsts.Timeout, testconsts.Interval).Should(Equal(metricType))
}

var _ = Describe("Validate knative functionality", func() {
	It("Should create a ksvc with cpu metric annotation when creating a capp with cpu scale metric", func() {
		testMetricAnnotation("cpu")
	})

	It("Should create a ksvc with memory metric annotation when creating a capp with memory scale metric", func() {
		testMetricAnnotation("memory")
	})

	It("Should create a ksvc with rps metric annotation when creating a capp with rps scale metric", func() {
		testMetricAnnotation("rps")
	})

	It("Should create and delete a ksvc when creating and deleting a capp instance", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		testCapp.Spec.ScaleMetric = "cpu"
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)
		assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)

		By("Checking if the ksvc was created successfully")
		ksvcObject := mocks.CreateKnativeServiceObject(assertionCapp.Name)
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, ksvcObject)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "Should find a resource.")
		checkRevisionReadiness(assertionCapp.Name+testconsts.FirstRevisionSuffix, true)

		By("Checking the ksvc has the needed labels")
		ksvcObject = utilst.GetKSVC(k8sClient, assertionCapp.Name, mocks.NSName)
		Expect(ksvcObject.Labels[testconsts.CappResourceKey]).Should(Equal(assertionCapp.Name))
		Expect(ksvcObject.Labels[testconsts.ManagedByLabelKey]).Should(Equal(testconsts.CappKey))

		By("Deleting the capp instance")
		utilst.DeleteCapp(k8sClient, assertionCapp)
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, assertionCapp)
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(BeTrue(), "Should not find a resource.")

		By("Checking if the ksvc exists")
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, ksvcObject)
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(BeTrue(), "Should not find a resource.")

		By("Checking if the revision exists")
		revisionObject := mocks.CreateRevisionObject(assertionCapp.Name + testconsts.FirstRevisionSuffix)
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, revisionObject)
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(BeTrue(), "Should not find a resource.")
	})

	It("Should update ksvc metric annotation and create a new revision when updating the capp scale metric", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		testCapp.Spec.ScaleMetric = "cpu"
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)

		By("Updating the Capp scale metric")
		var latestReadyRevisionBeforeUpdate string
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Spec.ScaleMetric = "memory"
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(true, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)

		By("Checking if the ksvc was updated successfully")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, createdCapp.Name, createdCapp.Namespace)
			return ksvc.Spec.Template.Annotations[testconsts.KnativeMetricAnnotation]
		}, testconsts.Timeout, testconsts.Interval).Should(Equal("memory"))
	})

	It("Should update ksvc container name and create a new revision when updating a capp container name", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)

		By("Updating the a capp container name")
		var latestReadyRevisionBeforeUpdate string
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Name = testconsts.TestContainerName
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(true, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)

		By("Checking if the ksvc was updated successfully")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, createdCapp.Name, createdCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Spec.Containers[0].Name
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(testconsts.TestContainerName))
	})

	It("Should update ksvc container image and create a new revision when updating a capp container image", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)

		By("Updating capp's container image")
		var latestReadyRevisionBeforeUpdate string
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Image = testconsts.ImageExample
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(true, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)

		By("Checking if the ksvc's container image was updated successfully")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, createdCapp.Name, createdCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Spec.Containers[0].Image
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(testconsts.ImageExample))
	})

	It("Should update ksvc dana annotation when updating capp's dana annotation", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)

		By("Updating capp's dana annotation")
		cappAnnotations := map[string]string{}
		cappAnnotations[testconsts.ExampleDanaAnnotation] = testconsts.ExampleAppName

		var latestReadyRevisionBeforeUpdate string
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Annotations = cappAnnotations
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(true, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)

		By("Checking if the ksvc's dana annotation was updated successfully")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, createdCapp.Name, createdCapp.Namespace)
			return ksvc.Spec.Template.Annotations[testconsts.ExampleDanaAnnotation]
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(testconsts.ExampleAppName))
	})

	It("Should update ksvc environment variable and create a new revision when updating a capp container environment variable", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)

		By("Updating capp's container environment variable")
		var latestReadyRevisionBeforeUpdate string
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env[0].Value = testconsts.ExampleAppName
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(true, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)

		By("Checking if the ksvc's container environment variable was updated successfully")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, createdCapp.Name, createdCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env[0].Value
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(testconsts.ExampleAppName))
	})

	It("Should create a new revision in ready state when updating capp's secret environment variable", func() {
		By("Creating a secret")
		secretName := utilst.GenerateSecretName()
		secretObject := mocks.CreateSecretObject(secretName)
		utilst.CreateSecret(k8sClient, secretObject)

		By("Creating a capp instance with a secret environment variable")
		testCapp := mocks.CreateBaseCapp()
		testCapp.Spec.ConfigurationSpec.Template.Spec.PodSpec.Containers[0].Env = *mocks.CreateEnvVarObject(secretName)
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)
		assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)

		checkRevisionReadiness(assertionCapp.Name+testconsts.FirstRevisionSuffix, true)

		By("Updating the secret")
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			secretObject := utilst.GetSecret(k8sClient, secretObject.Name, secretObject.Namespace)
			secretObject.Data = map[string][]byte{testconsts.NewSecretKey: []byte(mocks.SecretValue)}

			return utilst.UpdateResource(k8sClient, secretObject)
		})
		Expect(err).To(BeNil())

		By("Updating the capp secret environment variable")
		var latestReadyRevisionBeforeUpdate string
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Key = testconsts.NewSecretKey
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(true, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)
	})

	It("Should create not ready revision when attempting to update to non existing image", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)
		assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)

		By("Updating capp's container image")
		var latestReadyRevisionBeforeUpdate string
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Image = testconsts.NonExistingImageExample
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(false, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)

		By("Checking if the ksvc's container image was updated successfully")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Spec.Containers[0].Image
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(testconsts.NonExistingImageExample))
	})

	It("Should create not ready revision when attempting to update to non existing secret", func() {
		By("Creating a secret")
		secretName := utilst.GenerateSecretName()
		secretObject := mocks.CreateSecretObject(secretName)
		utilst.CreateSecret(k8sClient, secretObject)

		By("Creating a capp instance with a secret environment variable")
		testCapp := mocks.CreateBaseCapp()
		testCapp.Spec.ConfigurationSpec.Template.Spec.PodSpec.Containers[0].Env = *mocks.CreateEnvVarObject(secretName)
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)

		checkRevisionReadiness(createdCapp.Name+testconsts.FirstRevisionSuffix, true)

		By("Updating the capp secret environment variable")
		nonExistingSecretName := utilst.GenerateSecretName()

		var latestReadyRevisionBeforeUpdate string
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)
			latestReadyRevisionBeforeUpdate = assertionCapp.Status.KnativeObjectStatus.ConfigurationStatusFields.LatestReadyRevisionName

			assertionCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env = *mocks.CreateEnvVarObject(nonExistingSecretName)
			return utilst.UpdateResource(k8sClient, assertionCapp)
		})
		Expect(err).To(BeNil())

		verifyLatestReadyRevision(false, createdCapp.Name, createdCapp.Namespace, latestReadyRevisionBeforeUpdate)
	})

	It("Should create capp with autoscale annotation. The default annotation in the ksvc should be overridden", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		annotations := map[string]string{
			testconsts.KnativeAutoscaleTargetKey: "666",
		}
		testCapp.Spec.ConfigurationSpec.Template.Annotations = annotations
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)
		assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)

		By("Checking if the ksvc's defaults annotations were overridden")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Annotations[testconsts.KnativeAutoscaleTargetKey]
		}, testconsts.Timeout, testconsts.Interval).Should(Equal("666"))
	})

	It("Should propagate Capp labels to the underlying KSVC", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		labels := map[string]string{
			testconsts.TestLabelKey:    "test",
			testconsts.CappResourceKey: "test",
		}
		testCapp.ObjectMeta.Labels = labels
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)
		assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)

		By("Checking if user-defined labels were propagated to the ksvc")
		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Labels[testconsts.TestLabelKey]
		}, testconsts.Timeout, testconsts.Interval).Should(Equal("test"))

		By("Checking if labels set by the controller cannot be overridden by users")
		Consistently(func() string {
			ksvc := utilst.GetKSVC(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Labels[testconsts.CappResourceKey]
		}, testconsts.DefaultConsistently, testconsts.Interval).ShouldNot(Equal("test"))

		Eventually(func() string {
			ksvc := utilst.GetKSVC(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.Labels[testconsts.CappResourceKey]
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(assertionCapp.Name))
	})

	It("Should check the default ksvc annotation is equal to the cappConfig's concurrency value", func() {
		By("Creating a capp instance")
		testCapp := mocks.CreateBaseCapp()
		createdCapp := utilst.CreateCapp(k8sClient, testCapp)
		assertionCapp := utilst.GetCapp(k8sClient, createdCapp.Name, createdCapp.Namespace)

		cappConfig := utilst.GetCappConfig(k8sClient, mocks.CappConfigName, mocks.ControllerNS)

		By("Checking if the ksvc's annotation is equal to the cappConfig's autoScale")
		Eventually(func() bool {
			ksvc := utilst.GetKSVC(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return ksvc.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations[testconsts.KnativeAutoscaleTargetKey] == fmt.Sprintf("%d", cappConfig.Spec.AutoscaleConfig.Concurrency) &&
				ksvc.Spec.ConfigurationSpec.Template.ObjectMeta.Annotations[testconsts.KnativeActivationScaleKey] == fmt.Sprintf("%d", cappConfig.Spec.AutoscaleConfig.ActivationScale)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())
	})
})
