package localdata

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Account struct {
	Name     string
	Password string
	Aliases  []string
}

type Character struct {
	Account string
	Name    string
}

type Store struct {
	AccountsPath   string
	CharactersPath string
	Accounts       []Account
	Characters     []Character
}

func (s *Store) Load() error {
	if err := s.loadAccounts(); err != nil {
		return err
	}
	return s.loadCharacters()
}

func (s *Store) loadAccounts() error {
	f, err := os.Open(s.AccountsPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Accounts = nil
			return nil
		}
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}
	var out []Account
	for i, row := range rows {
		if len(row) < 2 {
			continue
		}
		if i == 0 && strings.EqualFold(row[0], "name") {
			continue
		}
		a := Account{Name: strings.TrimSpace(row[0]), Password: row[1]}
		a.Aliases = []string{a.Name}
		if len(row) >= 3 && row[2] != "" {
			for _, al := range strings.Split(row[2], "|") {
				al = strings.TrimSpace(al)
				if al != "" {
					a.Aliases = append(a.Aliases, al)
				}
			}
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	s.Accounts = out
	return nil
}

func (s *Store) loadCharacters() error {
	f, err := os.Open(s.CharactersPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Characters = nil
			return nil
		}
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		s.Characters = nil
		return nil
	}
	start := 0
	legacy := false
	if strings.EqualFold(rows[0][0], "account") {
		start = 1
	} else if strings.EqualFold(rows[0][0], "name") && len(rows[0]) >= 2 && strings.Contains(strings.ToLower(rows[0][1]), "account") {
		start = 1
		legacy = true
	}
	var out []Character
	for _, row := range rows[start:] {
		if len(row) < 2 {
			continue
		}
		if legacy {
			out = append(out, Character{Name: strings.TrimSpace(row[0]), Account: strings.TrimSpace(row[1])})
		} else {
			out = append(out, Character{Account: strings.TrimSpace(row[0]), Name: strings.TrimSpace(row[1])})
		}
	}
	s.Characters = out
	return nil
}

func (s *Store) SaveAccounts() error {
	if err := os.MkdirAll(filepath.Dir(s.AccountsPath), 0o700); err != nil {
		return err
	}
	sort.Slice(s.Accounts, func(i, j int) bool {
		return strings.ToLower(s.Accounts[i].Name) < strings.ToLower(s.Accounts[j].Name)
	})
	tmp := s.AccountsPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"name", "password", "aliases"})
	for _, a := range s.Accounts {
		extra := make([]string, 0)
		for _, al := range a.Aliases {
			if !strings.EqualFold(al, a.Name) {
				extra = append(extra, al)
			}
		}
		_ = w.Write([]string{a.Name, a.Password, strings.Join(extra, "|")})
	}
	w.Flush()
	if err := f.Close(); err != nil {
		return err
	}
	bak := s.AccountsPath + ".bak"
	if _, err := os.Stat(s.AccountsPath); err == nil {
		_ = os.Rename(s.AccountsPath, bak)
	}
	return os.Rename(tmp, s.AccountsPath)
}

func (s *Store) SaveCharacters() error {
	if err := os.MkdirAll(filepath.Dir(s.CharactersPath), 0o700); err != nil {
		return err
	}
	tmp := s.CharactersPath + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"account", "name"})
	for _, c := range s.Characters {
		_ = w.Write([]string{c.Account, c.Name})
	}
	w.Flush()
	if err := f.Close(); err != nil {
		return err
	}
	bak := s.CharactersPath + ".bak"
	if _, err := os.Stat(s.CharactersPath); err == nil {
		_ = os.Rename(s.CharactersPath, bak)
	}
	return os.Rename(tmp, s.CharactersPath)
}

// UpsertAccount adds or updates by account name (case-insensitive). Empty password keeps existing.
func (s *Store) UpsertAccount(name, password string, aliases []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("account name required")
	}
	cleanAliases := make([]string, 0, len(aliases)+1)
	seen := map[string]bool{strings.ToLower(name): true}
	cleanAliases = append(cleanAliases, name)
	for _, al := range aliases {
		al = strings.TrimSpace(al)
		if al == "" || seen[strings.ToLower(al)] {
			continue
		}
		seen[strings.ToLower(al)] = true
		cleanAliases = append(cleanAliases, al)
	}
	for i := range s.Accounts {
		if strings.EqualFold(s.Accounts[i].Name, name) {
			if password != "" {
				s.Accounts[i].Password = password
			}
			s.Accounts[i].Name = name
			s.Accounts[i].Aliases = cleanAliases
			return s.SaveAccounts()
		}
	}
	if password == "" {
		return fmt.Errorf("password required for new account")
	}
	s.Accounts = append(s.Accounts, Account{Name: name, Password: password, Aliases: cleanAliases})
	return s.SaveAccounts()
}

func (s *Store) DeleteAccount(name string) error {
	name = strings.TrimSpace(name)
	out := s.Accounts[:0]
	found := false
	for _, a := range s.Accounts {
		if strings.EqualFold(a.Name, name) {
			found = true
			continue
		}
		out = append(out, a)
	}
	if !found {
		return fmt.Errorf("account not found")
	}
	s.Accounts = out
	// Drop characters for removed account
	chars := s.Characters[:0]
	for _, c := range s.Characters {
		if !strings.EqualFold(c.Account, name) {
			chars = append(chars, c)
		}
	}
	s.Characters = chars
	if err := s.SaveAccounts(); err != nil {
		return err
	}
	return s.SaveCharacters()
}

func (s *Store) UpsertCharacter(account, name string) error {
	account = strings.TrimSpace(account)
	name = strings.TrimSpace(name)
	if account == "" || name == "" {
		return fmt.Errorf("account and character name required")
	}
	for i := range s.Characters {
		if strings.EqualFold(s.Characters[i].Name, name) {
			s.Characters[i] = Character{Account: account, Name: name}
			return s.SaveCharacters()
		}
	}
	s.Characters = append(s.Characters, Character{Account: account, Name: name})
	return s.SaveCharacters()
}

func (s *Store) DeleteCharacter(name string) error {
	name = strings.TrimSpace(name)
	out := s.Characters[:0]
	found := false
	for _, c := range s.Characters {
		if strings.EqualFold(c.Name, name) {
			found = true
			continue
		}
		out = append(out, c)
	}
	if !found {
		return fmt.Errorf("character not found")
	}
	s.Characters = out
	return s.SaveCharacters()
}

type ResolveResult struct {
	Matched    bool
	ViaAlias   bool
	Candidates []Account
	AllBusy    bool
	Chosen     *Account
	Error      string
}

func (s *Store) ResolveLogin(typed string, busy map[string]bool) ResolveResult {
	typed = strings.TrimSpace(typed)

	for _, c := range s.Characters {
		if strings.EqualFold(c.Name, typed) {
			typed = c.Account
			break
		}
	}

	var cands []Account
	viaAlias := false
	for _, a := range s.Accounts {
		if strings.EqualFold(a.Name, typed) {
			cands = append(cands, a)
			continue
		}
		for _, al := range a.Aliases {
			if strings.EqualFold(al, typed) {
				cands = append(cands, a)
				viaAlias = true
				break
			}
		}
	}
	if len(cands) == 0 {
		return ResolveResult{Matched: false}
	}
	if len(cands) > 1 {
		viaAlias = true
	}

	// Direct account / character login: never block on "online". EQ will report
	// already-logged-in if the session is truly occupied.
	if len(cands) == 1 {
		chosen := cands[0]
		return ResolveResult{Matched: true, ViaAlias: viaAlias, Candidates: cands, Chosen: &chosen}
	}

	// Multi-account alias pool: rotate to a free account (same idea as SSO tags).
	var free []Account
	for _, a := range cands {
		if !busy[strings.ToLower(a.Name)] {
			free = append(free, a)
		}
	}
	if len(free) == 0 {
		return ResolveResult{Matched: true, ViaAlias: true, Candidates: cands, AllBusy: true}
	}
	chosen := free[0]
	for _, a := range free[1:] {
		if strings.ToLower(a.Name) < strings.ToLower(chosen.Name) {
			chosen = a
		}
	}
	return ResolveResult{Matched: true, ViaAlias: true, Candidates: cands, Chosen: &chosen}
}

func DefaultPaths(dir string) (accounts, characters string) {
	return filepath.Join(dir, "local_accounts.csv"), filepath.Join(dir, "local_characters.csv")
}

// ParseAccountsCSV reads name,password[,aliases] rows from path (same format as local store).
func ParseAccountsCSV(path string) ([]Account, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	var out []Account
	for i, row := range rows {
		if len(row) < 2 {
			continue
		}
		if i == 0 && strings.EqualFold(strings.TrimSpace(row[0]), "name") {
			continue
		}
		name := strings.TrimSpace(row[0])
		if name == "" {
			continue
		}
		a := Account{Name: name, Password: row[1], Aliases: []string{name}}
		if len(row) >= 3 && row[2] != "" {
			seen := map[string]bool{strings.ToLower(name): true}
			for _, al := range strings.Split(row[2], "|") {
				al = strings.TrimSpace(al)
				if al == "" || seen[strings.ToLower(al)] {
					continue
				}
				seen[strings.ToLower(al)] = true
				a.Aliases = append(a.Aliases, al)
			}
		}
		out = append(out, a)
	}
	return out, nil
}

// ImportAccountsCSV merges accounts from path into the store (by name, case-insensitive).
// Existing accounts are updated (password + aliases); new ones are added. Saves once.
func (s *Store) ImportAccountsCSV(path string) (added, updated int, err error) {
	incoming, err := ParseAccountsCSV(path)
	if err != nil {
		return 0, 0, err
	}
	if len(incoming) == 0 {
		return 0, 0, fmt.Errorf("no accounts found in file")
	}
	for _, in := range incoming {
		found := false
		for i := range s.Accounts {
			if !strings.EqualFold(s.Accounts[i].Name, in.Name) {
				continue
			}
			found = true
			if in.Password != "" {
				s.Accounts[i].Password = in.Password
			}
			s.Accounts[i].Name = in.Name
			s.Accounts[i].Aliases = in.Aliases
			updated++
			break
		}
		if found {
			continue
		}
		if in.Password == "" {
			return added, updated, fmt.Errorf("password required for new account %q", in.Name)
		}
		s.Accounts = append(s.Accounts, in)
		added++
	}
	if err := s.SaveAccounts(); err != nil {
		return added, updated, err
	}
	return added, updated, nil
}

// ExportAccountsCSV writes all accounts to path in import-compatible format
// (name,password,aliases with | -separated aliases).
func (s *Store) ExportAccountsCSV(path string) (int, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, fmt.Errorf("path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return 0, err
	}
	accounts := append([]Account(nil), s.Accounts...)
	sort.Slice(accounts, func(i, j int) bool {
		return strings.ToLower(accounts[i].Name) < strings.ToLower(accounts[j].Name)
	})
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}
	w := csv.NewWriter(f)
	if err := w.Write([]string{"name", "password", "aliases"}); err != nil {
		_ = f.Close()
		return 0, err
	}
	for _, a := range accounts {
		extra := make([]string, 0)
		for _, al := range a.Aliases {
			if !strings.EqualFold(al, a.Name) {
				extra = append(extra, al)
			}
		}
		if err := w.Write([]string{a.Name, a.Password, strings.Join(extra, "|")}); err != nil {
			_ = f.Close()
			return 0, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	return len(accounts), nil
}

