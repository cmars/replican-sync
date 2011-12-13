package sqlite3

import (
	"log"
	"os"
	"path/filepath"

	"github.com/cmars/replican-sync/replican/fs"
	"github.com/kuroneko/gosqlite3"
)

var tbl_BLOCKS *sqlite3.Table
var tbl_FILES *sqlite3.Table
var tbl_DIRS *sqlite3.Table

func init() {
	tbl_BLOCKS = &sqlite3.Table{ "blocks", "parent INTEGER, strong TEXT, weak INTEGER, pos INTEGER" }
	tbl_FILES = &sqlite3.Table{ "files", "parent INTEGER, strong TEXT, name TEXT, mode INTEGER, size INTEGER" }
	tbl_DIRS = &sqlite3.Table{ "dirs", "parent INTEGER, strong TEXT, name TEXT, mode INTEGER" }
}

type DbRepo struct {
	RootPath string

	chanRoot      chan *cmdRoot
	chanWeakBlock chan *cmdWeakBlock
	chanBlock     chan *cmdBlock
	chanFile      chan *cmdFile
	chanDir       chan *cmdDir
	chanAddBlock  chan *cmdAddBlock
	chanAddFile   chan *cmdAddFile
	chanAddDir    chan *cmdAddDir
	
	chanParentOf chan *cmdParentOf
	chanSubdirsOf chan *cmdSubdirsOf
	chanFilesOf chan *cmdFilesOf
	chanBlocksOf chan *cmdBlocksOf
	chanUpdateStrong chan *cmdUpdateStrong
	
	exit chan bool
}

type dbBlock struct {
	id int64
	parent int64
	repo *DbRepo
	info *fs.BlockInfo
}

func (dbb *dbBlock) Repo() fs.NodeRepo { return dbb.repo }

func (dbb *dbBlock)	Parent() (fs.FsNode, bool) {
	return dbb.repo.parentOf(dbb)
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
	return dbf.repo.parentOf(dbf)
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
	return dbf.repo.blocksOf(dbf)
}

type dbDir struct {
	id int64
	parent int64
	repo *DbRepo
	info *fs.DirInfo
}

func (dbd *dbDir) Repo() fs.NodeRepo { return dbd.repo }

func (dbd *dbDir) Parent() (fs.FsNode, bool) {
	return dbd.repo.parentOf(dbd)
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
	return dbd.repo.subdirsOf(dbd)
}

func (dbd *dbDir) Files() []fs.File {
	return dbd.repo.filesOf(dbd)
}

func (dbd *dbDir) UpdateStrong() string {
	return dbd.repo.updateStrong(dbd)
}

func (dbRepo *DbRepo) doRoot(db *sqlite3.Database) fs.FsNode {
	stmt, _ := db.Prepare("SELECT rowid, strong, name, mode FROM dirs WHERE parent = NULL")
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

func (dbRepo *DbRepo) doWeakBlock(db *sqlite3.Database, weak int) (fs.Block, bool) {
	stmt, _ := db.Prepare(
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

func (dbRepo *DbRepo) doBlock(db *sqlite3.Database, strong string) (fs.Block, bool) {
	stmt, _ := db.Prepare(
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

func (dbRepo *DbRepo) doFile(db *sqlite3.Database, strong string) (fs.File, bool) {
	stmt, _ := db.Prepare(
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

func (dbRepo *DbRepo) doDir(db *sqlite3.Database, strong string) (fs.Dir, bool) {
	stmt, _ := db.Prepare(
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

func (dbRepo *DbRepo) doAddBlock(db *sqlite3.Database, file fs.File, blockInfo *fs.BlockInfo) fs.Block {
	dbfile := file.(*dbFile)
	stmt, _ := db.Prepare(
		`INSERT INTO blocks (parent, strong, weak, pos) VALUES (?,?,?,?)`, 
		dbfile.id, blockInfo.Strong, blockInfo.Weak, blockInfo.Position)
	stmt.Step()
	stmt.Finalize()
	
	stmt, _ = db.Prepare(`SELECT last_insert_rowid()`)
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

func (dbRepo *DbRepo) doAddFile(db *sqlite3.Database, dir fs.Dir, fileInfo *fs.FileInfo, blocksInfo []*fs.BlockInfo) fs.File {
	dbdir := dir.(*dbDir)
	stmt, _ := db.Prepare(
		`INSERT INTO files (parent, strong, name, mode, size) VALUES (?,?,?,?,?)`, 
		dbdir.id, fileInfo.Strong, fileInfo.Name, fileInfo.Mode, fileInfo.Size)
	stmt.Step()
	stmt.Finalize()
	
	stmt, _ = db.Prepare(`SELECT last_insert_rowid()`)
	stmt.Step()
	values := stmt.Row()
	stmt.Finalize()
	file := &dbFile{
		repo: dbRepo,
		id: values[0].(int64),
		parent: dbdir.id,
		info: fileInfo }
	
	for _, blockInfo := range blocksInfo {
		dbRepo.doAddBlock(db, file, blockInfo)
	}
	
	return file
}

func (dbRepo *DbRepo) doAddDir(db *sqlite3.Database, dir fs.Dir, subdirInfo *fs.DirInfo) fs.Dir {
	var id int64
	var stmt *sqlite3.Statement
	var err os.Error
	sql := `INSERT INTO dirs (parent, strong, name, mode) VALUES (?1,?2,?3,?4)`
	if dbdir, is := dir.(*dbDir); is {
		id = dbdir.id
		stmt, err = db.Prepare(sql, 
			dbdir.id, subdirInfo.Strong, subdirInfo.Name, int(subdirInfo.Mode))
	} else {
		id = int64(-1)
		stmt, err = db.Prepare(sql,
			nil, subdirInfo.Strong, subdirInfo.Name, int(subdirInfo.Mode))
//			nil, subdirInfo.Strong, subdirInfo.Name, int(subdirInfo.Mode))
	}
	if err != nil { log.Printf("%v\n", err) }
	stmt.Step()
	stmt.Finalize()
	
	stmt, _ = db.Prepare(`SELECT last_insert_rowid()`)
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

func (dbRepo *DbRepo) doParentOf(db *sqlite3.Database, node fs.Node) (fs.FsNode, bool) {
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
		
		stmt, _ := db.Prepare(sql, id)
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
	
	stmt, _ := db.Prepare(sql, id)
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

func (dbRepo *DbRepo) doSubdirsOf(db *sqlite3.Database, dir *dbDir) []fs.Dir {
	result := []fs.Dir{}
	stmt, _ := db.Prepare(
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

func (dbRepo *DbRepo) doFilesOf(db *sqlite3.Database, dir *dbDir) []fs.File {
	result := []fs.File{}
	stmt, _ := db.Prepare(
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

func (dbRepo *DbRepo) doBlocksOf(db *sqlite3.Database, file *dbFile) []fs.Block {
	result := []fs.Block{}
	stmt, _ := db.Prepare(
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

func (dbRepo *DbRepo) doUpdateStrong(db *sqlite3.Database, dir *dbDir) string {
	newStrong := fs.CalcStrong(dir)
	if newStrong != dir.info.Strong {
		stmt, _ := db.Prepare(
			`UPDATE dirs SET strong = ? WHERE id = ?`, newStrong, dir.id)
		defer stmt.Finalize()
		stmt.Step()
	}
	return newStrong
}

func NewDbRepo(rootPath string) *DbRepo {
	dbRepo := &DbRepo{
		RootPath:      rootPath,
		
		chanRoot:      make(chan *cmdRoot),
		chanWeakBlock: make(chan *cmdWeakBlock),
		chanBlock:     make(chan *cmdBlock),
		chanFile:      make(chan *cmdFile),
		chanDir:       make(chan *cmdDir),
		chanAddBlock:  make(chan *cmdAddBlock),
		chanAddFile:   make(chan *cmdAddFile),
		chanAddDir:    make(chan *cmdAddDir),
		
	chanParentOf: make(chan *cmdParentOf),
	chanSubdirsOf: make(chan *cmdSubdirsOf),
	chanFilesOf: make(chan *cmdFilesOf),
	chanBlocksOf: make(chan *cmdBlocksOf),
	chanUpdateStrong: make(chan *cmdUpdateStrong),
	
		exit:          make(chan bool, 1)}
	go dbRepo.run()
	return dbRepo
}

func (dbRepo *DbRepo) dbPath() string {
	return filepath.Join(dbRepo.RootPath, ".replican/db")
}

func (dbRepo *DbRepo) requireTables(db *sqlite3.Database) {
	_, err := db.Execute(`CREATE TABLE IF NOT EXISTS blocks (
		parent INTEGER,
		strong TEXT,
		weak INTEGER,
		pos INTEGER)`)
	if err != nil { log.Printf("%v", err) }
	_, err = db.Execute(`CREATE TABLE IF NOT EXISTS files (
		parent INTEGER,
		strong TEXT,
		name TEXT,
		mode INTEGER,
		size INTEGER)`)
	if err != nil { log.Printf("%v", err) }
	_, err = db.Execute(`CREATE TABLE IF NOT EXISTS dirs (
		parent INTEGER,
		strong TEXT,
		name TEXT,
		mode INTEGER)`)
	if err != nil { log.Printf("%v", err) }
}

func (dbRepo *DbRepo) run() {
	dbDir, _ := filepath.Split(dbRepo.dbPath())
	os.MkdirAll(dbDir, 0755)
	sqlite3.Initialize()
	defer sqlite3.Shutdown()
	
	log.Printf("database file: %v", dbRepo.dbPath())
	db, err := sqlite3.Open(dbRepo.dbPath())
	if err != nil { log.Printf("%v", err) }
	dbRepo.requireTables(db)
	
RUNNING:
	for {
		select {
		case cmd := <-dbRepo.chanRoot:
			cmd.result = dbRepo.doRoot(db)
			dbRepo.chanRoot <- cmd
		
		case cmd := <-dbRepo.chanWeakBlock:
			cmd.resBlock, cmd.resHas = dbRepo.doWeakBlock(db, cmd.argWeak)
			dbRepo.chanWeakBlock <- cmd
		
		case cmd := <-dbRepo.chanBlock:
			cmd.resBlock, cmd.resHas = dbRepo.doBlock(db, cmd.argStrong)
			dbRepo.chanBlock <- cmd
		
		case cmd := <-dbRepo.chanFile:
			cmd.resFile, cmd.resHas = dbRepo.doFile(db, cmd.argStrong)
			dbRepo.chanFile <- cmd
		
		case cmd := <-dbRepo.chanDir:
			cmd.resDir, cmd.resHas = dbRepo.doDir(db, cmd.argStrong)
			dbRepo.chanDir <- cmd
		
		case cmd := <-dbRepo.chanAddBlock:
			cmd.result = dbRepo.doAddBlock(db, cmd.argFile, cmd.argBlockInfo)
			dbRepo.chanAddBlock <- cmd
		
		case cmd := <-dbRepo.chanAddFile:
			cmd.result = dbRepo.doAddFile(db, cmd.argDir, cmd.argFileInfo, cmd.argBlocksInfo)
			dbRepo.chanAddFile <- cmd
		
		case cmd := <-dbRepo.chanAddDir:
			cmd.result = dbRepo.doAddDir(db, cmd.argDir, cmd.argSubDirInfo)
			dbRepo.chanAddDir <- cmd
	
		case cmd := <-dbRepo.chanParentOf:
			cmd.resNode, cmd.resHas = dbRepo.doParentOf(db, cmd.arg)
			dbRepo.chanParentOf <- cmd
		
		case cmd := <-dbRepo.chanSubdirsOf:
			cmd.result = dbRepo.doSubdirsOf(db, cmd.arg)
			dbRepo.chanSubdirsOf <- cmd
	
		case cmd := <-dbRepo.chanFilesOf:
			cmd.result = dbRepo.doFilesOf(db, cmd.arg)
			dbRepo.chanFilesOf <- cmd
	
		case cmd := <-dbRepo.chanBlocksOf:
			cmd.result = dbRepo.doBlocksOf(db, cmd.arg)
			dbRepo.chanBlocksOf <- cmd
	
		case cmd := <-dbRepo.chanUpdateStrong:
			cmd.result = dbRepo.doUpdateStrong(db, cmd.arg)
			dbRepo.chanUpdateStrong <- cmd
		
		case _ = <-dbRepo.exit:
			break RUNNING
		
		}
	}
	close(dbRepo.exit)
}

/*
 * Commands and channel stubs implementing NodeRepo
 */

type cmdRoot struct {
	result fs.FsNode
}

func (dbRepo *DbRepo) Root() fs.FsNode {
	cmd := &cmdRoot{}
	dbRepo.chanRoot <- cmd
	cmd = <-dbRepo.chanRoot
	return cmd.result
}

type cmdWeakBlock struct {
	argWeak  int
	resBlock fs.Block
	resHas   bool
}

func (dbRepo *DbRepo) WeakBlock(weak int) (fs.Block, bool) {
	cmd := &cmdWeakBlock{argWeak: weak}
	dbRepo.chanWeakBlock <- cmd
	cmd = <-dbRepo.chanWeakBlock
	return cmd.resBlock, cmd.resHas
}

type cmdBlock struct {
	argStrong string
	resBlock  fs.Block
	resHas    bool
}

func (dbRepo *DbRepo) Block(strong string) (fs.Block, bool) {
	cmd := &cmdBlock{argStrong: strong}
	dbRepo.chanBlock <- cmd
	cmd = <-dbRepo.chanBlock
	return cmd.resBlock, cmd.resHas
}

type cmdFile struct {
	argStrong string
	resFile   fs.File
	resHas    bool
}

func (dbRepo *DbRepo) File(strong string) (fs.File, bool) {
	cmd := &cmdFile{argStrong: strong}
	dbRepo.chanFile <- cmd
	cmd = <-dbRepo.chanFile
	return cmd.resFile, cmd.resHas
}

type cmdDir struct {
	argStrong string
	resDir    fs.Dir
	resHas    bool
}

func (dbRepo *DbRepo) Dir(strong string) (fs.Dir, bool) {
	cmd := &cmdDir{argStrong: strong}
	dbRepo.chanDir <- cmd
	cmd = <-dbRepo.chanDir
	return cmd.resDir, cmd.resHas
}

type cmdAddBlock struct {
	argFile      fs.File
	argBlockInfo *fs.BlockInfo
	result       fs.Block
}

func (dbRepo *DbRepo) AddBlock(file fs.File, blockInfo *fs.BlockInfo) fs.Block {
	cmd := &cmdAddBlock{argFile: file, argBlockInfo: blockInfo}
	dbRepo.chanAddBlock <- cmd
	cmd = <-dbRepo.chanAddBlock
	return cmd.result
}

type cmdAddFile struct {
	argDir        fs.Dir
	argFileInfo   *fs.FileInfo
	argBlocksInfo []*fs.BlockInfo
	result        fs.File
}

func (dbRepo *DbRepo) AddFile(dir fs.Dir, fileInfo *fs.FileInfo, blocksInfo []*fs.BlockInfo) fs.File {
	cmd := &cmdAddFile{argDir: dir, argFileInfo: fileInfo, argBlocksInfo: blocksInfo}
	dbRepo.chanAddFile <- cmd
	cmd = <-dbRepo.chanAddFile
	return cmd.result
}

type cmdAddDir struct {
	argDir        fs.Dir
	argSubDirInfo *fs.DirInfo
	result        fs.Dir
}

func (dbRepo *DbRepo) AddDir(dir fs.Dir, subdirInfo *fs.DirInfo) fs.Dir {
	cmd := &cmdAddDir{argDir: dir, argSubDirInfo: subdirInfo}
	dbRepo.chanAddDir <- cmd
	cmd = <-dbRepo.chanAddDir
	return cmd.result
}

type cmdParentOf struct {
	arg fs.Node
	resNode fs.FsNode
	resHas bool
}

func (dbRepo *DbRepo) parentOf(node fs.Node) (fs.FsNode, bool) {
	cmd := &cmdParentOf{arg: node}
	dbRepo.chanParentOf <- cmd
	cmd = <-dbRepo.chanParentOf
	return cmd.resNode, cmd.resHas
}

type cmdSubdirsOf struct {
	arg *dbDir
	result []fs.Dir
}

func (dbRepo *DbRepo) subdirsOf(dir *dbDir) []fs.Dir {
	cmd := &cmdSubdirsOf{arg: dir}
	dbRepo.chanSubdirsOf <- cmd
	cmd = <-dbRepo.chanSubdirsOf
	return cmd.result
}

type cmdFilesOf struct {
	arg *dbDir
	result []fs.File
}

func (dbRepo *DbRepo) filesOf(dir *dbDir) []fs.File {
	cmd := &cmdFilesOf{arg: dir}
	dbRepo.chanFilesOf <- cmd
	cmd = <-dbRepo.chanFilesOf
	return cmd.result
}

type cmdBlocksOf struct {
	arg *dbFile
	result []fs.Block
}

func (dbRepo *DbRepo) blocksOf(file *dbFile) []fs.Block {
	cmd := &cmdBlocksOf{arg: file}
	dbRepo.chanBlocksOf <- cmd
	cmd = <-dbRepo.chanBlocksOf
	return cmd.result
}

type cmdUpdateStrong struct {
	arg *dbDir
	result string
}

func (dbRepo *DbRepo) updateStrong(dir *dbDir) string {
	cmd := &cmdUpdateStrong{arg: dir}
	dbRepo.chanUpdateStrong <- cmd
	cmd = <-dbRepo.chanUpdateStrong
	return cmd.result
}
