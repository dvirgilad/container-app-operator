package testconsts

import (
	"time"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
)

const (
	Timeout             = 300 * time.Second
	Interval            = 2 * time.Second
	DefaultEventually   = 2 * time.Second
	DefaultConsistently = 30 * time.Second
)

const (
	UnsupportedScaleMetric    = "storage"
	EnabledState              = "enabled"
	DisabledState             = "disabled"
	KnativeMetricAnnotation   = "autoscaling.knative.dev/metric"
	ImageExample              = "danateam/autoscale-go"
	NonExistingImageExample   = "example-python-app:v1"
	ExampleAppName            = "new-app-name"
	NewSecretKey              = "username"
	ExampleDanaAnnotation     = "rcs.dana.io/app-name"
	TestContainerName         = "capp-test-container"
	FirstRevisionSuffix       = "-00001"
	KnativeAutoscaleTargetKey = "autoscaling.knative.dev/target"
	KnativeActivationScaleKey = "autoscaling.knative.dev/activation-scale"
	TestIndex                 = "test"
	TestLabelKey              = "e2e-test"
)

var (
	CappAPIGroup      = cappv1alpha1.GroupVersion.Group
	CappNamespaceKey  = CappAPIGroup + "/parent-capp-ns"
	CappResourceKey   = CappAPIGroup + "/parent-capp"
	ManagedByLabelKey = CappAPIGroup + "/managed-by"
)

const (
	CappKey = "capp"
)
