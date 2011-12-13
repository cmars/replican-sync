package sqlite3

import (
	"log"
	"os"

	"github.com/cmars/replican-sync/replican/fs"
	"github.com/kuroneko/gosqlite3"
)

type DbRepo struct {
	RootPath string
	db *sqlite3.Database
}

type dbBlock struct {
	id int64
	parent int64
	repo *DbRepo
	info *fs.BlockInfo
}

func (dbb *dbBlock) Repo() fs.NodeRepo { return dbb.repo }

func (dbb *dbBlock)	Parent() (fs.FsNode, bool) {
	return dbb.repo.ParentOf(dbb)
}
	
func (dbb *dbBlock) Info() *fs.BlockInfo {
	return dbb.info
}
	
type dbFile struct {
	id int64
	parent int64
	repo *DbRepo
	info *fs.FileInfo
}

func (dbf *dbFile) Repo() fs.NodeRepo { return dbf.repo }

func (dbf *dbFile) Parent() (fs.FsNode, bool) {
	return dbf.repo.ParentOf(dbf)
}
	
func (dbf *dbFile) Info() *fs.FileInfo {
	return dbf.info
}
	
func (dbf *dbFile) Name() string {
	return dbf.info.Name
}

func (dbf *dbFile) Mode() uint32 {
	return dbf.info.Mode
}

func (dbf *dbFile) Blocks() []fs.Block {
	return dbf.repo.BlocksOf(dbf)
}

type dbDir struct {
	id int64
	parent int64
	repo *DbRepo
	info *fs.DirInfo
}

func (dbd *dbDir) Repo() fs.NodeRepo { return dbd.repo }

func (dbd *dbDir) Parent() (fs.FsNode, bool) {
	return dbd.repo.ParentOf(dbd)
}
	
func (dbd *dbDir) Info() *fs.DirInfo {
	return dbd.info
}
	
func (dbd *dbDir) Name() string {
	return dbd.info.Name
}

func (dbd *dbDir) Mode() uint32 {
	return dbd.info.Mode
}

func (dbd *dbDir) SubDirs() []fs.Dir {
	return dbd.repo.SubdirsOf(dbd)
}

func (dbd *dbDir) Files() []fs.File {
	return dbd.repo.FilesOf(dbd)
}

func (dbd *dbDir) UpdateStrong() string {
	return dbd.repo.UpdateStrong(dbd)
}

func (dbRepo *DbRepo) Root() fs.FsNode {
	stmt, _ := dbRepo.db.Prepare(
		"SELECT rowid, strong, name, mode FROM dirs WHERE parent = NULL")
	defer stmt.Finalize()
	stmt.Step()
	values := stmt.Row()
	dir := &dbDir{
		repo: dbRepo,
		id: values[0].(int64),
		info: &fs.DirInfo {
			Strong: values[1].(string),
			Name: values[2].(string),
			Mode: values[3].(uint32) } }
	return dir
}

func (dbRepo *DbRepo) WeakBlock(weak int) (fs.Block, bool) {
	stmt, _ := dbRepo.db.Prepare(
		`SELECT b.rowid, p.rowid, b.pos, b.strong, p.strong 
			FROM blocks AS b LEFT OUTER JOIN files AS p ON b.parent = p.rowid
			WHERE b.weak = ?`, weak)
	defer stmt.Finalize()
	stmt.Step()
	values := stmt.Row()
	block := &dbBlock{
		repo: dbRepo,
		id: values[0].(int64),
		parent: values[1].(int64),
		info: &fs.BlockInfo {
			Weak: weak,
			Position: values[2].(int),
			Strong: values[3].(string),
			Parent: values[4].(string) } }
	return block, true
}

func (dbRepo *DbRepo) Block(strong string) (fs.Block, bool) {
	stmt, _ := dbRepo.db.Prepare(
		`SELECT b.rowid, p.rowid, b.weak, b.pos, p.strong 
			FROM blocks AS b LEFT OUTER JOIN files AS p ON b.parent = p.rowid
			WHERE b.strong = ?`, strong)
	defer stmt.Finalize()
	stmt.Step()
	values := stmt.Row()
	block := &dbBlock{
		repo: dbRepo,
		id: values[0].(int64),
		parent: values[1].(int64),
		info: &fs.BlockInfo {
			Weak: values[2].(int),
			Position: values[3].(int),
			Strong: strong,
			Parent: values[4].(string) } }
	return block, true
}

func (dbRepo *DbRepo) File(strong string) (fs.File, bool) {
	stmt, _ := dbRepo.db.Prepare(
		`SELECT f.rowid, p.rowid, f.name, f.mode, f.size, p.strong
			FROM files AS f LEFT OUTER JOIN dirs AS p ON f.parent = p.rowid
			WHERE f.strong = ?`, strong)
	defer stmt.Finalize()
	stmt.Step()
	values := stmt.Row()
	file := &dbFile{
		repo: dbRepo,
		id: values[0].(int64),
		parent: values[1].(int64),
		info: &fs.FileInfo {
			Strong: strong,
			Name: values[2].(string),
			Mode: values[3].(uint32),
			Size: values[4].(int64),
			Parent: values[5].(string) } }
	return file, true
}

func (dbRepo *DbRepo) Dir(strong string) (fs.Dir, bool) {
	stmt, _ := dbRepo.db.Prepare(
		`SELECT d.rowid, p.rowid, d.name, d.mode, p.strong 
			FROM dirs AS d LEFT OUTER JOIN dirs AS p ON d.parent = p.rowid
			WHERE d.strong = ?`, strong)
	defer stmt.Finalize()
	stmt.Step()
	values := stmt.Row()
	dir := &dbDir{ 
		repo: dbRepo,
		id: values[0].(int64),
		parent: values[1].(int64),
		info: &fs.DirInfo {
			Name: values[2].(string),
			Mode: values[3].(uint32),
			Strong: strong,
			Parent: values[4].(string) } }
	return dir, true
}

func (dbRepo *DbRepo) AddBlock(file fs.File, blockInfo *fs.BlockInfo) fs.Block {
	dbfile := file.(*dbFile)
	stmt, _ := dbRepo.db.Prepare(
		`INSERT INTO blocks (parent, strong, weak, pos) VALUES (?,?,?,?)`, 
		dbfile.id, blockInfo.Strong, blockInfo.Weak, blockInfo.Position)
	stmt.Step()
	stmt.Finalize()
	
	stmt, _ = dbRepo.db.Prepare(`SELECT last_insert_rowid()`)
	stmt.Step()
	values := stmt.Row()
	stmt.Finalize()
	block := &dbBlock{
		repo: dbRepo,
		id: values[0].(int64),
		parent: dbfile.id,
		info: blockInfo }
	return block
}

func (dbRepo *DbRepo) AddFile(dir fs.Dir, fileInfo *fs.FileInfo, blocksInfo []*fs.BlockInfo) fs.File {
	dbdir := dir.(*dbDir)
	stmt, _ := dbRepo.db.Prepare(
		`INSERT INTO files (parent, strong, name, mode, size) VALUES (?,?,?,?,?)`, 
		dbdir.id, fileInfo.Strong, fileInfo.Name, fileInfo.Mode, fileInfo.Size)
	stmt.Step()
	stmt.Finalize()
	
	stmt, _ = dbRepo.db.Prepare(`SELECT last_insert_rowid()`)
	stmt.Step()
	values := stmt.Row()
	stmt.Finalize()
	file := &dbFile{
		repo: dbRepo,
		id: values[0].(int64),
		parent: dbdir.id,
		info: fileInfo }
	
	for _, blockInfo := range blocksInfo {
		dbRepo.AddBlock(file, blockInfo)
	}
	
	return file
}

func (dbRepo *DbRepo) AddDir(dir fs.Dir, subdirInfo *fs.DirInfo) fs.Dir {
	var id int64
	var stmt *sqlite3.Statement
	var err os.Error
	sql := `INSERT INTO dirs (parent, strong, name, mode) VALUES (?1,?2,?3,?4)`
	if dbdir, is := dir.(*dbDir); is {
		id = dbdir.id
		stmt, err = dbRepo.db.Prepare(sql, 
			dbdir.id, subdirInfo.Strong, subdirInfo.Name, int(subdirInfo.Mode))
	} else {
		id = int64(-1)
		stmt, err = dbRepo.db.Prepare(sql,
			nil, subdirInfo.Strong, subdirInfo.Name, int(subdirInfo.Mode))
//			nil, subdirInfo.Strong, subdirInfo.Name, int(subdirInfo.Mode))
	}
	if err != nil { log.Printf("%v\n", err) }
	stmt.Step()
	stmt.Finalize()
	
	stmt, _ = dbRepo.db.Prepare(`SELECT last_insert_rowid()`)
	stmt.Step()
	values := stmt.Row()
	stmt.Finalize()
	subdir := &dbDir{
		repo: dbRepo,
		id: values[0].(int64),
		parent: id,
		info: subdirInfo }
	return subdir
}

func (dbRepo *DbRepo) ParentOf(node fs.Node) (fs.FsNode, bool) {
	var sql string
	var id int64
	switch node.(type) {
	case *dbBlock:
		id = node.(*dbBlock).parent
		if id == 0 {
			return nil, false
		}
		
		sql = `SELECT f.rowid, p.rowid, f.name, f.mode, f.size, f.strong, p.strong
			FROM files AS f LEFT OUTER JOIN dirs AS p ON f.parent = p.rowid
			WHERE f.rowid = ?`
		
		stmt, _ := dbRepo.db.Prepare(sql, id)
		stmt.Step()
		values := stmt.Row()
		stmt.Finalize()
		return &dbFile{
			repo: dbRepo,
			id: values[0].(int64),
			parent: values[1].(int64),
			info: &fs.FileInfo {
				Name: values[2].(string),
				Mode: values[3].(uint32),
				Size: values[4].(int64),
				Strong: values[5].(string),
				Parent: values[6].(string) } }, true
		
	case *dbFile:
		id = node.(*dbFile).parent
		sql = `SELECT d.rowid, p.rowid, d.name, d.mode, d.strong, p.strong 
			FROM dirs AS d LEFT OUTER JOIN dirs AS p ON d.parent = p.rowid 
			WHERE d.rowid = ?`
	case *dbDir:
		id = node.(*dbDir).parent
		sql = `SELECT d.rowid, p.rowid, d.name, d.mode, d.strong, p.strong 
			FROM dirs AS d LEFT OUTER JOIN dirs AS p ON d.parent = p.rowid 
			WHERE d.rowid = ?`
	}
	
	if id == 0 {
		return nil, false
	}
	
	stmt, _ := dbRepo.db.Prepare(sql, id)
	stmt.Step()
	values := stmt.Row()
	stmt.Finalize()
	return &dbDir{
		repo: dbRepo,
		id: values[0].(int64),
		parent: values[1].(int64),
		info: &fs.DirInfo {
			Name: values[2].(string),
			Mode: values[3].(uint32),
			Strong: values[4].(string),
			Parent: values[5].(string) } }, true
}

func (dbRepo *DbRepo) SubdirsOf(dir *dbDir) []fs.Dir {
	result := []fs.Dir{}
	stmt, _ := dbRepo.db.Prepare(
		`SELECT d.rowid, p.rowid, d.name, d.mode, d.strong, p.strong 
			FROM dirs AS d LEFT OUTER JOIN dirs AS p ON d.parent = p.rowid
			WHERE p.rowid = ?`, dir.id)
	defer stmt.Finalize()
	
	stmt.All(func (_ *sqlite3.Statement, values ...interface{}){
		result = append(result, &dbDir{
			repo: dbRepo,
			id: values[0].(int64),
			parent: values[1].(int64),
			info: &fs.DirInfo {
				Name: values[2].(string),
				Mode: values[3].(uint32),
				Strong: values[4].(string),
				Parent: values[5].(string) } })
	})
	return result
}

func (dbRepo *DbRepo) FilesOf(dir *dbDir) []fs.File {
	result := []fs.File{}
	stmt, _ := dbRepo.db.Prepare(
		`SELECT f.rowid, p.rowid, f.name, f.mode, f.size, f.strong, p.strong 
			FROM files AS f LEFT OUTER JOIN dirs AS p ON f.parent = p.rowid
			WHERE p.rowid = ?`, dir.id)
	defer stmt.Finalize()
	
	stmt.All(func (_ *sqlite3.Statement, values ...interface{}){
		result = append(result, &dbFile{
			repo: dbRepo,
			id: values[0].(int64),
			parent: values[1].(int64),
			info: &fs.FileInfo {
				Name: values[2].(string),
				Mode: values[3].(uint32),
				Size: values[4].(int64),
				Strong: values[5].(string),
				Parent: values[6].(string) } })
	})
	return result
}

func (dbRepo *DbRepo) BlocksOf(file *dbFile) []fs.Block {
	result := []fs.Block{}
	stmt, _ := dbRepo.db.Prepare(
		`SELECT b.rowid, p.rowid, b.weak, b.pos, b.strong, p.strong 
			FROM blocks AS b LEFT OUTER JOIN files AS p ON b.parent = p.rowid
			WHERE p.rowid = ?`, file.id)
	defer stmt.Finalize()
	
	stmt.All(func (_ *sqlite3.Statement, values ...interface{}){
		result = append(result, &dbBlock{
			repo: dbRepo,
			id: values[0].(int64),
			parent: values[1].(int64),
			info: &fs.BlockInfo {
				Weak: values[2].(int),
				Position: values[3].(int),
				Strong: values[4].(string),
				Parent: values[5].(string) } })
	})
	return result
}

func (dbRepo *DbRepo) UpdateStrong(dir *dbDir) string {
	newStrong := fs.CalcStrong(dir)
	if newStrong != dir.info.Strong {
		stmt, err := dbRepo.db.Prepare(
			`UPDATE dirs SET strong = ?1 WHERE rowid = ?2`, newStrong, dir.id)
		if err != nil {
			log.Printf("%v", err)
		}
		stmt.Step()
		stmt.Finalize()
	}
	return newStrong
}

func (dbRepo *DbRepo) Close() {
	dbRepo.db.Close()
	dbRepo.db = nil
}

func NewDbRepo(dbpath string) (*DbRepo, os.Error) {
	db, err := sqlite3.Open(dbpath)
	if err != nil {
		return nil, err
	}
	
	dbRepo := &DbRepo{ db: db }
	err = dbRepo.createTables()
	return dbRepo, err
}

const cr_blocks_sql = `CREATE TABLE IF NOT EXISTS blocks (
		parent INTEGER,
		strong TEXT,
		weak INTEGER,
		pos INTEGER)`
const cr_files_sql = `CREATE TABLE IF NOT EXISTS files (
		parent INTEGER,
		strong TEXT,
		name TEXT,
		mode INTEGER,
		size INTEGER)`
const cr_dirs_sql = `CREATE TABLE IF NOT EXISTS dirs (
		parent INTEGER,
		strong TEXT,
		name TEXT,
		mode INTEGER)`

func (dbRepo *DbRepo) createTables() os.Error {
	for _, sql := range []string{ cr_blocks_sql, cr_files_sql, cr_dirs_sql } {
		_, err := dbRepo.db.Execute(sql)
		if err != nil {
			return err
		}
	}
	return nil
}
