package store

// Store groups the application's repositories.
type Store struct {
	Users        *UserStore
	Workstations *WorkstationStore
	Sessions     *AuthStore
}

// New constructs repository adapters over the supplied database handle.
func New(db DBTX) *Store {
	return &Store{
		Users:        NewUserStore(db),
		Workstations: NewWorkstationStore(db),
		Sessions:     NewAuthStore(db),
	}
}
