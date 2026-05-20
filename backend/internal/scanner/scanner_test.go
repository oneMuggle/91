package scanner

import (
	"context"
	"database/sql"
	"io"
	"testing"
	"time"

	"github.com/video-site/backend/internal/catalog"
	"github.com/video-site/backend/internal/drives"
)

func TestRunPersistsRemoteThumbnailFromDriveEntry(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	drv := &scannerFakeDrive{
		entries: []drives.Entry{{
			ID:           "file-1",
			Name:         "clip.mp4",
			Size:         123,
			MimeType:     "video/mp4",
			ModTime:      time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
			ThumbnailURL: "https://thumbnail.example/clip.jpg",
		}},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	stats, err := sc.Run(ctx, "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if stats.Added != 1 {
		t.Fatalf("added = %d, want 1", stats.Added)
	}

	got, err := cat.GetVideo(ctx, "fake-drive-file-1")
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	if got.ThumbnailURL != "https://thumbnail.example/clip.jpg" {
		t.Fatalf("thumbnail = %q, want remote thumbnail", got.ThumbnailURL)
	}
}

func TestRunIgnoresZeroSizeVideoFiles(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	drv := &scannerFakeDrive{
		entries: []drives.Entry{{
			ID:       "empty-file",
			Name:     "empty.mp4",
			Size:     0,
			MimeType: "video/mp4",
			ModTime:  time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
		}},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	stats, err := sc.Run(ctx, "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if stats.Added != 0 {
		t.Fatalf("added = %d, want 0", stats.Added)
	}
	if _, err := cat.GetVideo(ctx, "fake-drive-empty-file"); err != sql.ErrNoRows {
		t.Fatalf("get zero-size video error = %v, want sql.ErrNoRows", err)
	}
}

func TestRunBackfillsRemoteThumbnailForExistingVideo(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	now := time.Now()
	if err := cat.UpsertVideo(ctx, &catalog.Video{
		ID:            "fake-drive-file-1",
		DriveID:       "drive",
		FileID:        "file-1",
		Title:         "Clip",
		PreviewStatus: "pending",
		PublishedAt:   now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("seed video: %v", err)
	}

	drv := &scannerFakeDrive{
		entries: []drives.Entry{{
			ID:           "file-1",
			Name:         "clip.mp4",
			Size:         123,
			MimeType:     "video/mp4",
			ModTime:      now,
			ThumbnailURL: "https://thumbnail.example/backfilled.jpg",
		}},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	stats, err := sc.Run(ctx, "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if stats.Added != 0 {
		t.Fatalf("added = %d, want 0", stats.Added)
	}

	got, err := cat.GetVideo(ctx, "fake-drive-file-1")
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	if got.ThumbnailURL != "https://thumbnail.example/backfilled.jpg" {
		t.Fatalf("thumbnail = %q, want backfilled remote thumbnail", got.ThumbnailURL)
	}
}

func TestRunReplacesExistingVideoTagsWithFixedFilenameTags(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	now := time.Now()
	if err := cat.UpsertVideo(ctx, &catalog.Video{
		ID:            "fake-drive-file-1",
		DriveID:       "drive",
		FileID:        "file-1",
		Title:         "Old",
		Tags:          []string{"sunny", "kenny"},
		PreviewStatus: "pending",
		PublishedAt:   now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("seed video: %v", err)
	}

	drv := &scannerFakeDrive{
		entries: []drives.Entry{{
			ID:      "file-1",
			Name:    "女大后入.mp4",
			Size:    123,
			ModTime: now,
		}},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	if _, err := sc.Run(ctx, ""); err != nil {
		t.Fatalf("scan: %v", err)
	}

	got, err := cat.GetVideo(ctx, "fake-drive-file-1")
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	want := []string{"后入", "女大"}
	if !sameStrings(got.Tags, want) {
		t.Fatalf("tags = %#v, want %#v", got.Tags, want)
	}
}

func TestRunAddsShortCollectionDirectoryAsTag(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})
	now := time.Now()
	for _, id := range []string{"existing-1", "existing-2"} {
		if err := cat.UpsertVideo(ctx, &catalog.Video{
			ID:          id,
			DriveID:     "drive",
			FileID:      id,
			Title:       "Existing",
			Category:    "sunny",
			PublishedAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			t.Fatalf("seed existing sunny video: %v", err)
		}
	}

	drv := &scannerTreeFakeDrive{
		entries: map[string][]drives.Entry{
			"root": {{
				ID:    "dir-1",
				Name:  "sunny",
				IsDir: true,
			}},
			"dir-1": {{
				ID:       "file-1",
				ParentID: "dir-1",
				Name:     "clip.mp4",
				Size:     123,
				ModTime:  now,
			}},
		},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	if _, err := sc.Run(ctx, ""); err != nil {
		t.Fatalf("scan: %v", err)
	}

	got, err := cat.GetVideo(ctx, "fake-drive-file-1")
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	if !sameStrings(got.Tags, []string{"sunny"}) {
		t.Fatalf("tags = %#v, want sunny", got.Tags)
	}
}

func TestRunMapsAVCodeDirectoryToAVTag(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})
	now := time.Now()
	for _, id := range []string{"existing-1", "existing-2"} {
		if err := cat.UpsertVideo(ctx, &catalog.Video{
			ID:          id,
			DriveID:     "drive",
			FileID:      id,
			Title:       "Existing",
			Category:    "cc-1750027",
			PublishedAt: now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			t.Fatalf("seed existing AV code video: %v", err)
		}
	}

	drv := &scannerTreeFakeDrive{
		entries: map[string][]drives.Entry{
			"root": {{
				ID:    "dir-1",
				Name:  "cc-1750027",
				IsDir: true,
			}},
			"dir-1": {{
				ID:       "file-1",
				ParentID: "dir-1",
				Name:     "clip.mp4",
				Size:     123,
				ModTime:  now,
			}},
		},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	if _, err := sc.Run(ctx, ""); err != nil {
		t.Fatalf("scan: %v", err)
	}

	got, err := cat.GetVideo(ctx, "fake-drive-file-1")
	if err != nil {
		t.Fatalf("get video: %v", err)
	}
	if !sameStrings(got.Tags, []string{"AV"}) {
		t.Fatalf("tags = %#v, want AV", got.Tags)
	}
}

func TestRunSkipsDuplicateFileHashes(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	now := time.Now()
	drv := &scannerFakeDrive{
		entries: []drives.Entry{
			{
				ID:      "file-1",
				Name:    "first.mp4",
				Size:    123,
				Hash:    "hash-same",
				ModTime: now,
			},
			{
				ID:      "file-2",
				Name:    "second.mp4",
				Size:    123,
				Hash:    "hash-same",
				ModTime: now,
			},
		},
	}
	addedIDs := []string{}
	sc := New(cat, drv, []string{".mp4"}, 5, func(v *catalog.Video) {
		addedIDs = append(addedIDs, v.ID)
	})

	stats, err := sc.Run(ctx, "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if stats.Added != 1 {
		t.Fatalf("added = %d, want 1", stats.Added)
	}
	if len(addedIDs) != 1 || addedIDs[0] != "fake-drive-file-1" {
		t.Fatalf("on new ids = %#v, want first file only", addedIDs)
	}

	items, total, err := cat.ListVideos(ctx, catalog.ListParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list videos: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("visible videos total=%d len=%d, want 1", total, len(items))
	}
	if items[0].FileID != "file-1" {
		t.Fatalf("visible file id = %q, want file-1", items[0].FileID)
	}
}

func TestRunSkipsDuplicateFileNamesWithSameSizeWhenHashesMissing(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	now := time.Now()
	drv := &scannerFakeDrive{
		entries: []drives.Entry{
			{
				ID:      "file-1",
				Name:    "same-name.mp4",
				Size:    123,
				ModTime: now,
			},
			{
				ID:      "file-2",
				Name:    "same-name.mp4",
				Size:    123,
				ModTime: now,
			},
			{
				ID:      "file-3",
				Name:    "same-name.mp4",
				Size:    456,
				ModTime: now,
			},
		},
	}
	addedIDs := []string{}
	sc := New(cat, drv, []string{".mp4"}, 5, func(v *catalog.Video) {
		addedIDs = append(addedIDs, v.ID)
	})

	stats, err := sc.Run(ctx, "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if stats.Added != 2 {
		t.Fatalf("added = %d, want 2", stats.Added)
	}
	wantAdded := []string{"fake-drive-file-1", "fake-drive-file-3"}
	if !sameStrings(addedIDs, wantAdded) {
		t.Fatalf("on new ids = %#v, want %#v", addedIDs, wantAdded)
	}
	if _, err := cat.GetVideo(ctx, "fake-drive-file-2"); err != sql.ErrNoRows {
		t.Fatalf("duplicate video lookup error = %v, want sql.ErrNoRows", err)
	}
}

func TestRunReportsSeenVideoFileIDsAndVisitedDirectories(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	drv := &scannerTreeFakeDrive{
		entries: map[string][]drives.Entry{
			"root": {
				{ID: "dir-1", Name: "Folder", IsDir: true},
				{ID: "root-file", Name: "root.mp4", Size: 123},
				{ID: "note", Name: "note.txt", Size: 123},
			},
			"dir-1": {
				{ID: "nested-file", ParentID: "dir-1", Name: "nested.mp4", Size: 456},
				{ID: "empty-video", ParentID: "dir-1", Name: "empty.mp4", Size: 0},
			},
		},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	stats, err := sc.Run(ctx, "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if _, ok := stats.SeenFileIDs["root-file"]; !ok {
		t.Fatalf("seen file ids = %#v, want root-file", stats.SeenFileIDs)
	}
	if _, ok := stats.SeenFileIDs["nested-file"]; !ok {
		t.Fatalf("seen file ids = %#v, want live non-empty videos", stats.SeenFileIDs)
	}
	if _, ok := stats.SeenFileIDs["note"]; ok {
		t.Fatalf("seen file ids = %#v, want non-video entries excluded", stats.SeenFileIDs)
	}
	if _, ok := stats.SeenFileIDs["empty-video"]; ok {
		t.Fatalf("seen file ids = %#v, want zero-size entries excluded", stats.SeenFileIDs)
	}
	if _, ok := stats.VisitedDirIDs["root"]; !ok {
		t.Fatalf("visited dir ids = %#v, want root", stats.VisitedDirIDs)
	}
	if _, ok := stats.VisitedDirIDs["dir-1"]; !ok {
		t.Fatalf("visited dir ids = %#v, want nested dir", stats.VisitedDirIDs)
	}
	if stats.Errors != 0 {
		t.Fatalf("errors = %d, want 0", stats.Errors)
	}
}

func TestRunExcludesP115MoviesDirectoryFromStatsAndCatalog(t *testing.T) {
	ctx := context.Background()
	cat, err := catalog.Open(t.TempDir() + "/catalog.db")
	if err != nil {
		t.Fatalf("open catalog: %v", err)
	}
	t.Cleanup(func() {
		if err := cat.Close(); err != nil {
			t.Fatalf("close catalog: %v", err)
		}
	})

	drv := &scannerTreeFakeDrive{
		kind: "p115",
		id:   "115",
		entries: map[string][]drives.Entry{
			"root": {
				{ID: "movies-dir", Name: "影视", IsDir: true},
				{ID: "normal-file", Name: "normal.mp4", Size: 123},
			},
			"movies-dir": {
				{ID: "movie-file", ParentID: "movies-dir", Name: "movie.mp4", Size: 456},
				{ID: "nested-dir", Name: "Nested", IsDir: true},
			},
			"nested-dir": {
				{ID: "nested-movie-file", ParentID: "nested-dir", Name: "nested.mp4", Size: 789},
			},
		},
	}
	sc := New(cat, drv, []string{".mp4"}, 5, nil)

	stats, err := sc.Run(ctx, "")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if stats.Scanned != 1 {
		t.Fatalf("scanned = %d, want only non-excluded file counted", stats.Scanned)
	}
	if stats.Added != 1 {
		t.Fatalf("added = %d, want only non-excluded file added", stats.Added)
	}
	if _, ok := stats.ExcludedFileIDs["movie-file"]; !ok {
		t.Fatalf("excluded file ids = %#v, want movie-file", stats.ExcludedFileIDs)
	}
	if _, ok := stats.ExcludedFileIDs["nested-movie-file"]; !ok {
		t.Fatalf("excluded file ids = %#v, want nested-movie-file", stats.ExcludedFileIDs)
	}
	if _, err := cat.GetVideo(ctx, "p115-115-movie-file"); err != sql.ErrNoRows {
		t.Fatalf("excluded direct movie get error = %v, want sql.ErrNoRows", err)
	}
	if _, err := cat.GetVideo(ctx, "p115-115-nested-movie-file"); err != sql.ErrNoRows {
		t.Fatalf("excluded nested movie get error = %v, want sql.ErrNoRows", err)
	}
	if _, err := cat.GetVideo(ctx, "p115-115-normal-file"); err != nil {
		t.Fatalf("normal video was not added: %v", err)
	}
}

type scannerFakeDrive struct {
	entries []drives.Entry
}

func (d *scannerFakeDrive) Kind() string { return "fake" }
func (d *scannerFakeDrive) ID() string   { return "drive" }
func (d *scannerFakeDrive) Init(context.Context) error {
	return nil
}
func (d *scannerFakeDrive) List(context.Context, string) ([]drives.Entry, error) {
	return d.entries, nil
}
func (d *scannerFakeDrive) Stat(context.Context, string) (*drives.Entry, error) {
	return nil, drives.ErrNotSupported
}
func (d *scannerFakeDrive) StreamURL(context.Context, string) (*drives.StreamLink, error) {
	return &drives.StreamLink{URL: "https://video.example/clip.mp4"}, nil
}
func (d *scannerFakeDrive) Upload(context.Context, string, string, io.Reader, int64) (string, error) {
	return "", drives.ErrNotSupported
}
func (d *scannerFakeDrive) EnsureDir(context.Context, string) (string, error) {
	return "", drives.ErrNotSupported
}
func (d *scannerFakeDrive) RootID() string { return "root" }

type scannerTreeFakeDrive struct {
	kind    string
	id      string
	entries map[string][]drives.Entry
}

func (d *scannerTreeFakeDrive) Kind() string {
	if d.kind != "" {
		return d.kind
	}
	return "fake"
}
func (d *scannerTreeFakeDrive) ID() string {
	if d.id != "" {
		return d.id
	}
	return "drive"
}
func (d *scannerTreeFakeDrive) Init(context.Context) error {
	return nil
}
func (d *scannerTreeFakeDrive) List(_ context.Context, parentID string) ([]drives.Entry, error) {
	return d.entries[parentID], nil
}
func (d *scannerTreeFakeDrive) Stat(context.Context, string) (*drives.Entry, error) {
	return nil, drives.ErrNotSupported
}
func (d *scannerTreeFakeDrive) StreamURL(context.Context, string) (*drives.StreamLink, error) {
	return &drives.StreamLink{URL: "https://video.example/clip.mp4"}, nil
}
func (d *scannerTreeFakeDrive) Upload(context.Context, string, string, io.Reader, int64) (string, error) {
	return "", drives.ErrNotSupported
}
func (d *scannerTreeFakeDrive) EnsureDir(context.Context, string) (string, error) {
	return "", drives.ErrNotSupported
}
func (d *scannerTreeFakeDrive) RootID() string { return "root" }
