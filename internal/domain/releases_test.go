package domain

import (
	"reflect"
	"sort"
	"testing"
)

func TestReleasesListTheNewestVersionFirstByNumberNotByText(t *testing.T) {
	t.Parallel()
	got := []string{
		"someone/Widget/v1.1.1",
		"someone/Widget/v1.10.1",
		"someone/Widget/v1.2.0",
		"someone/Widget/nightly",
		"someone/Widget/1.9",
		"other/Alpha/v0.1.0",
		"someone/Widget/v1.10.1.1",
	}
	sort.Slice(got, func(i, j int) bool { return ReleaseBefore(got[i], got[j]) })
	want := []string{
		"other/Alpha/v0.1.0",
		"someone/Widget/v1.10.1.1",
		"someone/Widget/v1.10.1",
		"someone/Widget/1.9",
		"someone/Widget/v1.2.0",
		"someone/Widget/v1.1.1",
		"someone/Widget/nightly",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v\nwant    %v", got, want)
	}
}

func TestReleasesThatAreNotVersionsFollowInNameOrder(t *testing.T) {
	t.Parallel()
	if !ReleaseBefore("a/b/beta", "a/b/nightly") || ReleaseBefore("a/b/nightly", "a/b/beta") {
		t.Error("two tags that are not versions are not in name order")
	}
	if !ReleaseBefore("a/b/1.0", "a/b/v1.0.0") || ReleaseBefore("a/b/v1.0.0", "a/b/1.0") {
		t.Error("two tags naming the same version are not settled by name")
	}
}

func TestReleaseLabelsFromOneRepositoryAreTheTagAlone(t *testing.T) {
	t.Parallel()
	got := ReleaseLabels([]string{"someone/Widget/v1.2.0", "Someone/widget/v1.1.0"})
	if want := []string{"v1.2.0", "v1.1.0"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("labels = %v, want %v", got, want)
	}
}

func TestReleaseLabelsFromSeveralRepositoriesCarryTheirNames(t *testing.T) {
	t.Parallel()
	got := ReleaseLabels([]string{"someone/Widget/v1.3.0", "someone/Gadget/v1.3.0", "a/b"})
	if want := []string{"Widget v1.3.0", "Gadget v1.3.0", "a/b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("labels = %v, want %v", got, want)
	}
}

func TestReleaseLabelsNameTheOwnerWhereTwoRepositoriesShareAName(t *testing.T) {
	t.Parallel()
	got := ReleaseLabels([]string{"someone/Widget/v1.0.0", "other/Widget/v1.0.0", "someone/Gadget/v2.0.0"})
	want := []string{"someone/Widget v1.0.0", "other/Widget v1.0.0", "Gadget v2.0.0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("labels = %v, want %v", got, want)
	}
}

func TestAReleaseKeyWithoutATagSortsAsARepository(t *testing.T) {
	t.Parallel()
	if !ReleaseBefore("a/b", "a/c/v1") {
		t.Error("a key with no tag is not ordered by its repository")
	}
}
