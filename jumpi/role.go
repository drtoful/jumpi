package jumpi

import (
	"encoding/json"
	"log"
	"regexp"
	"sync"
)

type Role struct {
	Name        string `json:"name"`
	UserRegex   string `json:"rex_user"`
	TargetRegex string `json:"rex_target"`

	rexUser   *regexp.Regexp
	rexTarget *regexp.Regexp
}

type RoleManager struct {
	roles map[string]*Role
	mutex *sync.Mutex
}

var (
	manager *RoleManager
)

func init() {
	manager = &RoleManager{}
}

func InitRoleManager(store *Store) {
	manager.roles = make(map[string]*Role)
	manager.mutex = &sync.Mutex{}

	log.Println("role_manager: startup, loading stored roles")
	vals, err := store.Scan(BucketRoles, "", 0, -1)
	if err != nil {
		log.Printf("role_manager: error in loading roles: %s\n", err.Error())
	}

	for _, r := range vals {
		var role Role
		if err := json.Unmarshal([]byte(r.Value), &role); err != nil {
			log.Printf("role_manager: unable to parse role '%s': %s\n", r.Key, err.Error())
			continue
		}

		if err := AddRole(&role); err != nil {
			log.Printf("role_manager: unable to parse role '%s': %s\n", r.Key, err.Error())
		}
	}
}

func AddRole(role *Role) error {
	if role.rexUser == nil {
		rex, err := regexp.Compile(role.UserRegex)
		if err != nil {
			return err
		}
		role.rexUser = rex
	}

	if role.rexTarget == nil {
		rex, err := regexp.Compile(role.TargetRegex)
		if err != nil {
			return err
		}
		role.rexTarget = rex
	}

	manager.mutex.Lock()
	manager.roles[role.Name] = role
	log.Printf("role_manager: added role '%s'\n", role.Name)
	manager.mutex.Unlock()
	return nil
}

func DeleteRole(role *Role) error {
	log.Printf("role_manager: removed role '%s'\n", role.Name)
	delete(manager.roles, role.Name)
	return nil
}

func CheckRole(user, target string) (bool, string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	for _, role := range manager.roles {
		if role.rexUser == nil || role.rexTarget == nil {
			continue
		}

		if role.rexUser.MatchString(user) && role.rexTarget.MatchString(target) {
			return true, role.Name
		}
	}

	return false, ""
}

func (r *Role) Store(store *Store) error {
	jdata, err := json.Marshal(r)
	if err != nil {
		return err
	}

	if err := AddRole(r); err != nil {
		return err
	}

	return store.Set(BucketRoles, r.Name, string(jdata))
}

func (r *Role) Delete(store *Store) error {
	if err := DeleteRole(r); err != nil {
		return err
	}
	return store.Delete(BucketRoles, r.Name)
}
