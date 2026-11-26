package metadata

import (
	"context"
	"errors"
	"testing"

	"dhwani/recognition/fingerprint"
)

func TestUploadOwnershipAndRemovalModeration(t *testing.T) {
	ctx := context.Background()
	store := NewMemory()
	owner, err := store.CreateUser(ctx, "Uploader", "uploader@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateUser(ctx, "Other", "other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	song, _, err := store.AddSong(ctx, owner.ID, Song{Title: "Track", Artist: "Artist", Duration: 30},
		[]fingerprint.Fingerprint{{Hash: 123, T: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddToCollection(ctx, other.ID, 2, song.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.AddHistory(ctx, other.ID, song.ID, 0.9); err != nil {
		t.Fatal(err)
	}

	uploads, err := store.ListUserUploads(ctx, owner.ID)
	if err != nil || len(uploads) != 1 || uploads[0].Song.ID != song.ID {
		t.Fatalf("owner uploads = %+v, err %v", uploads, err)
	}
	uploads, err = store.ListUserUploads(ctx, other.ID)
	if err != nil || len(uploads) != 0 {
		t.Fatalf("other user's uploads = %+v, err %v", uploads, err)
	}
	if err := store.RequestSongRemoval(ctx, other.ID, song.ID); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("non-owner request error = %v, want ErrNotOwner", err)
	}
	if err := store.RequestSongRemoval(ctx, owner.ID, song.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.RequestSongRemoval(ctx, owner.ID, song.ID); !errors.Is(err, ErrPendingRemoval) {
		t.Fatalf("duplicate request error = %v, want ErrPendingRemoval", err)
	}

	requests, err := store.ListRemovalRequests(ctx)
	if err != nil || len(requests) != 1 || requests[0].Song.ID != song.ID {
		t.Fatalf("removal requests = %+v, err %v", requests, err)
	}
	if err := store.ReviewRemovalRequest(ctx, other.ID, requests[0].ID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-admin review error = %v, want ErrNotFound", err)
	}
	store.SetAdmin(other.ID, true)
	if err := store.ReviewRemovalRequest(ctx, other.ID, requests[0].ID, false); err != nil {
		t.Fatal(err)
	}
	uploads, err = store.ListUserUploads(ctx, owner.ID)
	if err != nil || len(uploads) != 1 || uploads[0].RemovalRequestStatus != "rejected" {
		t.Fatalf("upload after rejected request = %+v, err %v", uploads, err)
	}
	if err := store.RequestSongRemoval(ctx, owner.ID, song.ID); err != nil {
		t.Fatalf("request after rejection: %v", err)
	}
	requests, err = store.ListRemovalRequests(ctx)
	if err != nil || len(requests) != 1 {
		t.Fatalf("pending requests after retry = %+v, err %v", requests, err)
	}
	if err := store.ReviewRemovalRequest(ctx, other.ID, requests[0].ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetSong(ctx, song.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("song after approval error = %v, want ErrNotFound", err)
	}
	if matches, err := store.Lookup(ctx, []uint32{123}); err != nil || len(matches[123]) != 0 {
		t.Fatalf("fingerprints after approval = %+v, err %v", matches, err)
	}
	saved, err := store.CollectionSongs(ctx, other.ID, 0)
	if err != nil || len(saved) != 0 {
		t.Fatalf("saved songs after approval = %+v, err %v", saved, err)
	}
	history, err := store.ListHistory(ctx, other.ID, 10)
	if err != nil || len(history) != 0 {
		t.Fatalf("history after approval = %+v, err %v", history, err)
	}
}
