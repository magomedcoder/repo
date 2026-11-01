package migrate

import (
	"bufio"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

var migrationFileRE = regexp.MustCompile(`^(\d{4})_(.+)\.(up|down)\.sql$`)

type sqlMeta struct {
	Destructive bool
	GoBefore    string
	GoAfter     string
}

func LoadSQL(fsys fs.FS, dir string, hooks map[string]func(*gorm.DB) error) ([]Migration, error) {
	if hooks == nil {
		hooks = map[string]func(*gorm.DB) error{}
	}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}

	type pair struct {
		up   string
		down string
	}

	files := map[string]*pair{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		m := migrationFileRE.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}

		key := m[1] + "_" + m[2]
		p := files[key]
		if p == nil {
			p = &pair{}
			files[key] = p
		}

		switch m[3] {
		case "up":
			p.up = e.Name()
		case "down":
			p.down = e.Name()
		}
	}

	out := make([]Migration, 0, len(files))
	seen := map[int]string{}
	for key, p := range files {
		if p.up == "" {
			return nil, fmt.Errorf("migration %s: missing .up.sql", key)
		}

		parts := strings.SplitN(key, "_", 2)
		version, err := strconv.Atoi(parts[0])
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("migration %s: invalid version", key)
		}

		name := parts[1]
		if prev, ok := seen[version]; ok {
			return nil, fmt.Errorf("duplicate migration version %d (%s and %s)", version, prev, name)
		}

		seen[version] = name

		upBody, err := fs.ReadFile(fsys, dir+"/"+p.up)
		if err != nil {
			return nil, err
		}

		meta, sqlUp := parseHeader(string(upBody))

		var downFn func(*gorm.DB) error
		if p.down != "" {
			downBody, err := fs.ReadFile(fsys, dir+"/"+p.down)
			if err != nil {
				return nil, err
			}

			_, sqlDown := parseHeader(string(downBody))
			downSQL := sqlDown
			downFn = func(db *gorm.DB) error {
				return ExecScript(db, downSQL)
			}
		}

		before, after := meta.GoBefore, meta.GoAfter
		upSQL := sqlUp
		upFn := func(db *gorm.DB) error {
			if before != "" {
				hook, ok := hooks[before]
				if !ok {
					return fmt.Errorf("unknown go_before hook %q", before)
				}

				if err := hook(db); err != nil {
					return fmt.Errorf("go_before %s: %w", before, err)
				}
			}

			if err := ExecScript(db, upSQL); err != nil {
				return err
			}

			if after != "" {
				hook, ok := hooks[after]
				if !ok {
					return fmt.Errorf("unknown go_after hook %q", after)
				}

				if err := hook(db); err != nil {
					return fmt.Errorf("go_after %s: %w", after, err)
				}
			}

			return nil
		}

		out = append(out, Migration{
			Version:     version,
			Name:        name,
			Destructive: meta.Destructive,
			Up:          upFn,
			Down:        downFn,
		})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	for i, m := range out {
		if m.Version != i+1 {
			return nil, fmt.Errorf("migration versions must be contiguous starting at 1; gap at %d (found %d_%s)", i+1, m.Version, m.Name)
		}
	}

	return out, nil
}

func parseHeader(body string) (sqlMeta, string) {
	var meta sqlMeta
	var sql strings.Builder
	sc := bufio.NewScanner(strings.NewReader(body))
	inHeader := true
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if inHeader {
			if after, ok := strings.CutPrefix(trim, "--"); ok {
				key, val, ok := strings.Cut(strings.TrimSpace(after), ":")
				if ok {
					key = strings.ToLower(strings.TrimSpace(key))
					val = strings.TrimSpace(val)
					switch key {
					case "destructive":
						meta.Destructive = val == "true" || val == "1" || val == "yes"
					case "go_before":
						meta.GoBefore = val
					case "go_after":
						meta.GoAfter = val
					}
				}
				continue
			}
			if trim == "" {
				continue
			}

			inHeader = false
		}
		sql.WriteString(line)
		sql.WriteByte('\n')
	}

	return meta, strings.TrimSpace(sql.String())
}

func ExecScript(db *gorm.DB, script string) error {
	for _, stmt := range splitSQL(script) {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("%w\nstatement: %s", err, stmt)
		}
	}

	return nil
}

func splitSQL(script string) []string {
	var out []string
	var b strings.Builder
	for line := range strings.SplitSeq(script, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") {
			continue
		}

		b.WriteString(line)
		b.WriteByte('\n')
	}

	for part := range strings.SplitSeq(b.String(), ";") {
		stmt := strings.TrimSpace(part)
		if stmt != "" {
			out = append(out, stmt)
		}
	}

	return out
}
