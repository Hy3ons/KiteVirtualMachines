package account

import (
	"context"
	"strings"

	"github.com/google/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"kite/internal/auth"
	"kite/internal/store"
)

type ErrorKind string

const (
	ErrorKindInvalid  ErrorKind = "invalid"
	ErrorKindConflict ErrorKind = "conflict"
)

const defaultProfileImage = "base64encodedimage"

// RequestError describes a user request error returned by account service methods.
// Kind separates validation errors from duplicate user errors.
// Message is safe to return from kite-api responses.
// This type is used by HTTP handlers to map account failures to response status codes.
type RequestError struct {
	Kind    ErrorKind
	Message string
}

func (e RequestError) Error() string {
	return e.Message
}

// PublicUser is the frontend-safe user representation returned by kite-api.
// Password hashes are intentionally excluded.
// AccessLevel is copied from KiteUser spec.access_level.
// This type is used by account service methods and HTTP responses.
type PublicUser struct {
	Name         string `json:"name"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Namespace    string `json:"namespace"`
	ProfileImage string `json:"profile_image"`
	AccessLevel  int64  `json:"access_level"`
}

// SignUpRequest contains user-provided fields for creating a KiteUser CRD.
// Name is optional internal input; when empty, the service generates a stable metadata.name.
// Namespace is optional internal input; when empty, the service derives a namespace from metadata.name.
// Password is hashed before it is stored in KiteUser spec.password.
type SignUpRequest struct {
	Name      string
	Username  string
	Email     string
	Password  string
	Namespace string
}

// UpdateRequest contains admin-editable KiteUser spec fields.
// Nil fields are left unchanged.
// Password is hashed when provided.
// AccessLevel is validated against the known auth access level range.
type UpdateRequest struct {
	Email        *string
	Password     *string
	Namespace    *string
	ProfileImage *string
	AccessLevel  *int
}

// Service provides KiteUser account operations backed by Kubernetes CRDs.
// userStore reads and writes cluster-scoped KiteUser resources.
// passwordSalt is used to hash and verify KiteUser passwords.
// This service is called by kite-api handlers instead of embedding CRD logic in HTTP code.
type Service struct {
	userStore    *store.UserStore
	vmStore      *store.VirtualMachineStore
	passwordSalt string
}

// NewService creates an account service backed by a dynamic Kubernetes client.
// dynamicClient is used to create, read, update, and delete KiteUser CRDs.
// passwordSalt is used to hash signup passwords and verify login passwords.
// The returned service is used by kite-api authentication and user handlers.
func NewService(dynamicClient dynamic.Interface, passwordSalt string) *Service {
	return &Service{
		userStore:    store.NewUserStore(dynamicClient),
		vmStore:      store.NewVirtualMachineStore(dynamicClient),
		passwordSalt: passwordSalt,
	}
}

// Authenticate verifies an email and password against stored KiteUser data.
// ctx controls Kubernetes list calls.
// email is matched against KiteUser spec.email.
// password is compared to KiteUser spec.password after hashing with the configured salt.
// The returned boolean is false when credentials are invalid.
func (s *Service) Authenticate(ctx context.Context, email string, password string) (PublicUser, bool, error) {
	if s.passwordSalt == "" {
		return PublicUser{}, false, invalid("password salt is not configured")
	}

	user, ok, err := s.FindByEmail(ctx, email)
	if err != nil || !ok {
		return PublicUser{}, ok, err
	}

	spec, err := specFromObject(user)
	if err != nil {
		return PublicUser{}, false, err
	}

	storedPassword := stringValue(spec, "password")
	if !auth.VerifyPassword(password, s.passwordSalt, storedPassword) {
		return PublicUser{}, false, nil
	}
	if auth.PasswordNeedsRehash(storedPassword) {
		if err := s.rehashUserPassword(ctx, user, password); err != nil {
			return PublicUser{}, false, err
		}
	}

	return publicUserFromSpec(user.GetName(), spec), true, nil
}

// SignUp creates a KiteUser CRD for a public signup request.
// ctx controls Kubernetes API calls.
// req contains user-provided signup fields.
// The first existing user becomes admin; later signups start as read-only users.
func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (PublicUser, error) {
	record, err := s.newSignUpRecord(ctx, req)
	if err != nil {
		return PublicUser{}, err
	}

	created, err := s.userStore.Create(ctx, record)
	if apierrors.IsAlreadyExists(err) {
		return PublicUser{}, conflict("kite user already exists")
	}
	if err != nil {
		return PublicUser{}, err
	}

	return publicUserFromObject(created)
}

// List returns every KiteUser as frontend-safe response data.
// ctx controls the Kubernetes list call.
// The returned users omit password hashes.
// This method is used by manager and admin user list endpoints.
func (s *Service) List(ctx context.Context) ([]PublicUser, error) {
	list, err := s.userStore.List(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]PublicUser, 0, len(list.Items))
	for _, item := range list.Items {
		user, err := publicUserFromObject(&item)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	return users, nil
}

// Get reads one KiteUser by metadata.name.
// ctx controls the Kubernetes get call.
// name is metadata.name of the cluster-scoped KiteUser CRD.
// The returned user omits the stored password hash.
func (s *Service) Get(ctx context.Context, name string) (PublicUser, error) {
	user, err := s.userStore.Get(ctx, name)
	if err != nil {
		return PublicUser{}, err
	}

	return publicUserFromObject(user)
}

// Update changes admin-editable KiteUser spec fields.
// ctx controls Kubernetes get and update calls.
// name is metadata.name of the KiteUser CRD.
// req contains optional fields to change.
func (s *Service) Update(ctx context.Context, name string, req UpdateRequest) (PublicUser, error) {
	current, err := s.userStore.Get(ctx, name)
	if err != nil {
		return PublicUser{}, err
	}

	record, err := recordFromObject(current)
	if err != nil {
		return PublicUser{}, err
	}

	if err := s.rejectDuplicateEmailUpdate(ctx, record.Name, req.Email); err != nil {
		return PublicUser{}, err
	}
	if err := s.applyUpdate(&record, req); err != nil {
		return PublicUser{}, err
	}

	updated, err := s.userStore.Update(ctx, record)
	if err != nil {
		return PublicUser{}, err
	}

	return publicUserFromObject(updated)
}

// Delete removes one KiteUser CRD and its child KiteVirtualMachine CRDs.
// ctx controls Kubernetes get, list, and delete calls.
// name is metadata.name of the cluster-scoped KiteUser.
// Child KiteVirtualMachine resources are selected from KiteUser spec.namespace before the user is deleted.
func (s *Service) Delete(ctx context.Context, name string) error {
	user, err := s.userStore.Get(ctx, name)
	if err != nil {
		return err
	}

	spec, err := specFromObject(user)
	if err != nil {
		return err
	}

	if namespace := strings.TrimSpace(stringValue(spec, "namespace")); namespace != "" {
		if err := s.deleteVirtualMachinesInNamespace(ctx, namespace); err != nil {
			return err
		}
	}

	return s.userStore.Delete(ctx, name)
}

// rehashUserPassword rewrites a verified legacy password hash with the current bcrypt format.
// ctx controls the Kubernetes update call.
// user is the KiteUser object that authenticated successfully.
// password is the verified plain text password received by Authenticate.
// This function is used only after legacy hash verification succeeds.
func (s *Service) rehashUserPassword(ctx context.Context, user *unstructured.Unstructured, password string) error {
	record, err := recordFromObject(user)
	if err != nil {
		return err
	}
	passwordHash, err := auth.HashPassword(password, s.passwordSalt)
	if err != nil {
		return err
	}
	record.Spec.Password = passwordHash

	_, err = s.userStore.Update(ctx, record)
	return err
}

// deleteVirtualMachinesInNamespace deletes every KiteVirtualMachine CRD in one namespace.
// ctx controls Kubernetes list and delete calls.
// namespace is usually KiteUser spec.namespace.
// NotFound errors are ignored so repeated delete requests remain idempotent.
func (s *Service) deleteVirtualMachinesInNamespace(ctx context.Context, namespace string) error {
	list, err := s.vmStore.List(ctx, namespace)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, vm := range list.Items {
		err := s.vmStore.Delete(ctx, namespace, vm.GetName())
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) rejectDuplicateEmailUpdate(ctx context.Context, currentName string, email *string) error {
	if email == nil || emailLookupKey(*email) == "" {
		return nil
	}

	existingUser, found, err := s.FindByEmail(ctx, *email)
	if err != nil || !found {
		return err
	}
	if existingUser.GetName() != currentName {
		return conflict("email already exists")
	}

	return nil
}

// FindByUsername searches KiteUser CRDs by spec.username.
// ctx controls the Kubernetes list call.
// username is matched exactly after trimming surrounding whitespace.
// The returned boolean is false when no matching user exists.
func (s *Service) FindByUsername(ctx context.Context, username string) (*unstructured.Unstructured, bool, error) {
	username = strings.TrimSpace(username)
	list, err := s.userStore.List(ctx)
	if err != nil {
		return nil, false, err
	}

	for i := range list.Items {
		item := &list.Items[i]
		spec, err := specFromObject(item)
		if err != nil {
			continue
		}

		if stringValue(spec, "username") == username {
			return item, true, nil
		}
	}

	return nil, false, nil
}

// FindByEmail searches KiteUser CRDs by spec.email.
// ctx controls the Kubernetes list call.
// email is matched case-insensitively after trimming surrounding whitespace.
// The returned boolean is false when no matching user exists.
func (s *Service) FindByEmail(ctx context.Context, email string) (*unstructured.Unstructured, bool, error) {
	email = emailLookupKey(email)
	list, err := s.userStore.List(ctx)
	if err != nil {
		return nil, false, err
	}

	for i := range list.Items {
		item := &list.Items[i]
		spec, err := specFromObject(item)
		if err != nil {
			continue
		}

		if emailLookupKey(stringValue(spec, "email")) == email {
			return item, true, nil
		}
	}

	return nil, false, nil
}

// newSignUpRecord converts a signup request into a KiteUser store record.
// ctx controls the list call used to decide whether the signup is the first user.
// req contains the raw signup request values.
// The returned record is ready to be written as a KiteUser CRD.
func (s *Service) newSignUpRecord(ctx context.Context, req SignUpRequest) (store.KiteUserRecord, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Namespace = strings.TrimSpace(req.Namespace)

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return store.KiteUserRecord{}, invalid("username, email, and password are required")
	}
	if s.passwordSalt == "" {
		return store.KiteUserRecord{}, invalid("password salt is not configured")
	}
	if _, found, err := s.FindByUsername(ctx, req.Username); err != nil {
		return store.KiteUserRecord{}, err
	} else if found {
		return store.KiteUserRecord{}, conflict("username already exists")
	}
	if _, found, err := s.FindByEmail(ctx, req.Email); err != nil {
		return store.KiteUserRecord{}, err
	} else if found {
		return store.KiteUserRecord{}, conflict("email already exists")
	}

	accessLevel, err := s.accessLevelForNewUser(ctx)
	if err != nil {
		return store.KiteUserRecord{}, err
	}

	name := kiteUserName(req.Name)
	namespace := kiteUserNamespace(name, req.Namespace)
	passwordHash, err := auth.HashPassword(req.Password, s.passwordSalt)
	if err != nil {
		return store.KiteUserRecord{}, err
	}

	return store.KiteUserRecord{
		Name: name,
		Spec: store.KiteUserSpec{
			Username:     req.Username,
			Email:        req.Email,
			Password:     passwordHash,
			Namespace:    namespace,
			ProfileImage: defaultProfileImage,
			AccessLevel:  accessLevel,
		},
	}, nil
}

// kiteUserName returns the metadata.name used as the stable KiteUser primary key.
// requestedName is an optional internal override used by tests or future admin tooling.
// The returned name is Kubernetes DNS-compatible because it uses a fixed prefix plus a UUID.
// This helper is used during signup before writing the KiteUser CRD.
func kiteUserName(requestedName string) string {
	if requestedName != "" {
		return requestedName
	}

	return "ku-" + uuid.NewString()
}

// kiteUserNamespace returns the user namespace stored in KiteUser spec.namespace.
// userName is the stable KiteUser metadata.name.
// requestedNamespace is an optional internal override used by tests or future admin tooling.
// The returned namespace defaults to a PK-derived value so public signup cannot choose cluster namespaces.
func kiteUserNamespace(userName string, requestedNamespace string) string {
	if requestedNamespace != "" {
		return requestedNamespace
	}

	return "kite-user-" + userName
}

// accessLevelForNewUser returns the default access level for a signup.
// ctx controls the Kubernetes list call.
// The first KiteUser becomes admin.
// Every later KiteUser starts as read-only.
func (s *Service) accessLevelForNewUser(ctx context.Context) (int, error) {
	users, err := s.userStore.List(ctx)
	if err != nil {
		return 0, err
	}

	if len(users.Items) == 0 {
		return auth.AccessLevelAdmin, nil
	}

	return auth.AccessLevelReadOnly, nil
}

// applyUpdate applies an admin update request to a KiteUser record.
// record is modified in place before UserStore.Update writes it back.
// req contains optional user fields.
// Password values are hashed before being stored.
func (s *Service) applyUpdate(record *store.KiteUserRecord, req UpdateRequest) error {
	if req.Email != nil {
		record.Spec.Email = strings.TrimSpace(*req.Email)
		if record.Spec.Email == "" {
			return invalid("email cannot be empty")
		}
	}
	if req.Password != nil {
		if *req.Password == "" {
			return invalid("password cannot be empty")
		}
		if s.passwordSalt == "" {
			return invalid("password salt is not configured")
		}
		passwordHash, err := auth.HashPassword(*req.Password, s.passwordSalt)
		if err != nil {
			return err
		}
		record.Spec.Password = passwordHash
	}
	if req.Namespace != nil {
		record.Spec.Namespace = strings.TrimSpace(*req.Namespace)
		if record.Spec.Namespace == "" {
			return invalid("namespace cannot be empty")
		}
	}
	if req.ProfileImage != nil {
		record.Spec.ProfileImage = *req.ProfileImage
	}
	if req.AccessLevel != nil {
		if *req.AccessLevel < auth.AccessLevelReadOnly || *req.AccessLevel > auth.AccessLevelAdmin {
			return invalid("access_level must be between 0 and 3")
		}
		record.Spec.AccessLevel = *req.AccessLevel
	}

	return nil
}

// recordFromObject converts a KiteUser CRD object into a store record.
// obj is the unstructured object read from Kubernetes.
// The returned record preserves mutable spec fields for updates.
// This helper is used by Service.Update.
func recordFromObject(obj *unstructured.Unstructured) (store.KiteUserRecord, error) {
	spec, err := specFromObject(obj)
	if err != nil {
		return store.KiteUserRecord{}, err
	}

	return store.KiteUserRecord{
		Name: obj.GetName(),
		Spec: store.KiteUserSpec{
			Username:     stringValue(spec, "username"),
			Email:        stringValue(spec, "email"),
			Password:     stringValue(spec, "password"),
			Namespace:    stringValue(spec, "namespace"),
			ProfileImage: stringValue(spec, "profile_image"),
			AccessLevel:  int(intValue(spec, "access_level")),
		},
	}, nil
}

// publicUserFromObject converts a KiteUser CRD into frontend-safe response data.
// obj is the unstructured object returned by the dynamic Kubernetes client.
// The returned user excludes spec.password.
// This helper is used by Service methods that return user data.
func publicUserFromObject(obj *unstructured.Unstructured) (PublicUser, error) {
	spec, err := specFromObject(obj)
	if err != nil {
		return PublicUser{}, err
	}

	return publicUserFromSpec(obj.GetName(), spec), nil
}

// publicUserFromSpec converts KiteUser metadata and spec into PublicUser.
// name is metadata.name of the KiteUser CRD.
// spec is the unstructured spec field from Kubernetes.
// The returned user is safe to expose through HTTP responses.
func publicUserFromSpec(name string, spec map[string]any) PublicUser {
	return PublicUser{
		Name:         name,
		Username:     stringValue(spec, "username"),
		Email:        stringValue(spec, "email"),
		Namespace:    stringValue(spec, "namespace"),
		ProfileImage: stringValue(spec, "profile_image"),
		AccessLevel:  intValue(spec, "access_level"),
	}
}

// specFromObject reads the spec map from an unstructured KiteUser object.
// obj is the Kubernetes object returned by the dynamic client.
// The returned map contains KiteUser spec fields.
// An error means the object is malformed for the account service.
func specFromObject(obj *unstructured.Unstructured) (map[string]any, error) {
	spec, ok := obj.Object["spec"].(map[string]any)
	if !ok {
		return nil, invalid("invalid kite user spec")
	}

	return spec, nil
}

// stringValue reads one string field from an unstructured map.
// data is usually a KiteUser spec map.
// key is the field name to read.
// The returned value is empty when the field is missing or not a string.
func stringValue(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

func emailLookupKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// intValue reads one integer-like field from an unstructured map.
// data is usually a KiteUser spec map.
// key is the field name to read.
// The returned value is zero when the field is missing or not numeric.
func intValue(data map[string]any, key string) int64 {
	switch value := data[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func invalid(message string) error {
	return RequestError{Kind: ErrorKindInvalid, Message: message}
}

func conflict(message string) error {
	return RequestError{Kind: ErrorKindConflict, Message: message}
}

// RequestErrorKind returns the kind from a RequestError.
// err is any service error.
// The returned boolean is false when err is not a RequestError.
// This helper is used by HTTP handlers to choose status codes.
func RequestErrorKind(err error) (ErrorKind, bool) {
	requestErr, ok := err.(RequestError)
	if !ok {
		return "", false
	}

	return requestErr.Kind, true
}
