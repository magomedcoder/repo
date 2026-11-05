package usecase

import (
	"context"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

type memFolders struct {
	byPath map[string]*domain.Folder
}

func (m *memFolders) Create(*domain.Folder) error {
	return nil
}

func (m *memFolders) Update(*domain.Folder) error {
	return nil
}

func (m *memFolders) Delete(uint) error {
	return nil
}

func (m *memFolders) FindByID(uint) (*domain.Folder, error) {
	return nil, ErrFolderNotFound
}

func (m *memFolders) FindByPath(path string) (*domain.Folder, error) {
	if f, ok := m.byPath[path]; ok {
		cp := *f
		return &cp, nil
	}

	return nil, ErrFolderNotFound
}

func (m *memFolders) ListForMember(uint) ([]domain.Folder, error) {
	return nil, nil
}

func (m *memFolders) ExistsBySlug(string) (bool, error) {
	return false, nil
}

func TestNormalizeMemberRoleCapsStripsDeleteFolderForMember(t *testing.T) {
	t.Parallel()
	_, caps, err := normalizeMemberRoleCaps("member", map[string]bool{
		"view":           true,
		"manage_members": true,
		"delete_folder":  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if caps&domain.CapDeleteFolder != 0 {
		t.Fatalf("member must not keep delete_folder, caps=%#x", caps)
	}

	if caps&domain.CapManageMembers == 0 {
		t.Fatalf("custom manage_members should remain, caps=%#x", caps)
	}
}

func TestUpdateMemberAdminCannotPromoteToOwner(t *testing.T) {
	t.Parallel()
	folders := &memFolders{byPath: map[string]*domain.Folder{
		"org": {ID: 1, Path: "org"},
	}}
	members := &memMembers{}
	_ = members.Create(&domain.FolderMember{
		FolderID: 1,
		UserID:   1,
		Role:     domain.FolderRoleOwner,
		Caps:     domain.CapAll,
	})
	_ = members.Create(&domain.FolderMember{
		FolderID: 1,
		UserID:   2,
		Role:     domain.FolderRoleAdmin,
		Caps:     domain.CapsForRole(domain.FolderRoleAdmin),
	})
	_ = members.Create(&domain.FolderMember{
		FolderID: 1,
		UserID:   3,
		Role:     domain.FolderRoleMember,
		Caps:     domain.CapView,
	})

	users := &memUsers{
		items: []*domain.User{
			{
				ID:       3,
				Username: "target",
			},
		},
		next: 3,
	}
	authz := NewAuthz(members, nil, nil, nil, nil)
	uc := NewAccessUseCase(folders, nil, users, members, nil, nil, nil, nil, authz)

	_, err := uc.UpdateMember(context.Background(), UpsertMemberInput{
		ViewerID:   2,
		FolderPath: "org",
		Username:   "target",
		Role:       "owner",
	})
	if err != ErrFolderForbidden {
		t.Fatalf("want ErrFolderForbidden, got %v", err)
	}

	out, err := uc.UpdateMember(context.Background(), UpsertMemberInput{
		ViewerID:   2,
		FolderPath: "org",
		Username:   "target",
		Role:       "member",
		Caps: map[string]bool{
			"view":          true,
			"delete_folder": true,
		},
	})
	if err != nil {
		t.Fatalf("member custom caps should succeed after strip: %v", err)
	}

	if out.Caps["delete_folder"] {
		t.Fatal("delete_folder must be stripped from member caps")
	}
}

func TestUpdateMemberOwnerCanPromote(t *testing.T) {
	t.Parallel()
	folders := &memFolders{byPath: map[string]*domain.Folder{
		"org": {
			ID:   1,
			Path: "org",
		},
	}}

	members := &memMembers{}
	_ = members.Create(&domain.FolderMember{
		FolderID: 1,
		UserID:   1,
		Role:     domain.FolderRoleOwner,
		Caps:     domain.CapAll,
	})
	_ = members.Create(&domain.FolderMember{
		FolderID: 1,
		UserID:   3,
		Role:     domain.FolderRoleMember,
		Caps:     domain.CapView,
	})

	users := &memUsers{
		items: []*domain.User{
			{
				ID:       3,
				Username: "target",
			},
		},
		next: 3,
	}
	authz := NewAuthz(members, nil, nil, nil, nil)
	uc := NewAccessUseCase(folders, nil, users, members, nil, nil, nil, nil, authz)

	out, err := uc.UpdateMember(context.Background(), UpsertMemberInput{
		ViewerID:   1,
		FolderPath: "org",
		Username:   "target",
		Role:       "owner",
	})
	if err != nil {
		t.Fatal(err)
	}

	if out.Role != string(domain.FolderRoleOwner) {
		t.Fatalf("role=%s", out.Role)
	}
}
