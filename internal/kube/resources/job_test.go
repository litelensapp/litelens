package kubeResources

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	listersbatchv1 "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/cache"
)

type errorJobLister struct{ err error }

func (e *errorJobLister) List(_ labels.Selector) ([]*batchv1.Job, error) {
	return nil, e.err
}
func (e *errorJobLister) Jobs(_ string) listersbatchv1.JobNamespaceLister {
	return &errorJobNamespaceLister{e.err}
}

type errorJobNamespaceLister struct{ err error }

func (e *errorJobNamespaceLister) List(_ labels.Selector) ([]*batchv1.Job, error) {
	return nil, e.err
}
func (e *errorJobNamespaceLister) Get(_ string) (*batchv1.Job, error) {
	return nil, e.err
}

func newJobLister(jobs ...*batchv1.Job) listersbatchv1.JobLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, job := range jobs {
		_ = indexer.Add(job)
	}
	return listersbatchv1.NewJobLister(indexer)
}

func makeJob(name, namespace string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{Time: fixedTime},
		},
		Spec: batchv1.JobSpec{},
	}
}

func TestListJobs_SingleNamespace(t *testing.T) {
	job := makeJob("my-job", "production")
	lister := newJobLister(job)

	result, err := ListJobs(lister, "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Name != "my-job" {
		t.Errorf("Name = %q; want %q", result[0].Name, "my-job")
	}
}

func TestListJobs_EmptyNamespaceReturnsAll(t *testing.T) {
	job1 := makeJob("job-a", "ns-a")
	job2 := makeJob("job-b", "ns-b")
	lister := newJobLister(job1, job2)

	result, err := ListJobs(lister, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

func TestListJobs_SpecificNamespaceFilters(t *testing.T) {
	job1 := makeJob("job-a", "ns-a")
	job2 := makeJob("job-b", "ns-b")
	lister := newJobLister(job1, job2)

	result, err := ListJobs(lister, "ns-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Name != "job-a" {
		t.Errorf("Name = %q; want %q", result[0].Name, "job-a")
	}
}

func TestListJobs_EmptyLister_ReturnsEmptySlice(t *testing.T) {
	lister := newJobLister()

	result, err := ListJobs(lister, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil slice; got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty result; got %d items", len(result))
	}
}

func TestGetJobByName_Found(t *testing.T) {
	job := makeJob("my-job", "production")
	lister := newJobLister(job)

	result, err := GetJobByName(lister, "production", "my-job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "my-job" {
		t.Errorf("Name = %q; want %q", result.Name, "my-job")
	}
}

func TestGetJobByName_Age_NonZeroTimestamp(t *testing.T) {
	job := makeJob("my-job", "default")
	lister := newJobLister(job)

	result, err := GetJobByName(lister, "default", "my-job")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Age == "" {
		t.Error("Age should not be empty for a past CreationTimestamp")
	}
}

func TestToJob_ZeroTimestamp_AgeIsNonEmpty(t *testing.T) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job",
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{},
	}
	got := toJob(job)
	if got.Age == "" {
		t.Error("Age must not be empty for zero-value CreationTimestamp")
	}
}

func TestToJob_WithStatus(t *testing.T) {
	completions := int32(3)
	succeeded := int32(2)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-job",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: fixedTime},
		},
		Spec: batchv1.JobSpec{
			Completions: &completions,
		},
		Status: batchv1.JobStatus{
			Succeeded: succeeded,
		},
	}
	got := toJob(job)
	if got.Succeeded != 2 {
		t.Errorf("Succeeded = %d; want 2", got.Succeeded)
	}
}

func TestToJob_NoCompletions(t *testing.T) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-job",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: fixedTime},
		},
		Spec: batchv1.JobSpec{},
	}
	got := toJob(job)
	if got.Completions != 1 {
		t.Errorf("Completions = %d; want 1 (default)", got.Completions)
	}
}
