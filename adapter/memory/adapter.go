package memory

// Adapter is a no-op adapter that satisfies the auther.Adapter interface.
// All state is managed in-memory by the Authorizer itself.
type Adapter struct{}

// New creates a new in-memory adapter.
func New() *Adapter { return &Adapter{} }

func (a *Adapter) CreateRole(roleID, parentID string) error { return nil }
func (a *Adapter) DeleteRole(roleID string) error           { return nil }
func (a *Adapter) AllRoles() ([][2]string, error)           { return nil, nil }

func (a *Adapter) CreateUser(roleID, userID string) error { return nil }
func (a *Adapter) DeleteUser(userID string) error         { return nil }
func (a *Adapter) AllUsers() ([][2]string, error)         { return nil, nil }

func (a *Adapter) CreateGrant(srcRoleID, dstRoleID, resource string) error { return nil }
func (a *Adapter) DeleteGrant(srcRoleID, dstRoleID, resource string) error { return nil }
func (a *Adapter) AllGrants() ([][3]string, error)                         { return nil, nil }
