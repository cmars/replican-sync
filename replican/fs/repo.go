package fs

import (
	"fmt"
)

type NodeRepo interface {
	Root() FsNode

	CreateBlock(info *BlockInfo) Block

	CreateFile(info *FileInfo) File

	CreateDir(info *DirInfo) Dir

	GetWeakBlock(weak int) (Block, bool)

	GetBlock(strong string) (Block, bool)

	GetFile(strong string) (File, bool)

	GetDir(strong string) (Dir, bool)

	SetBlock(block Block)

	SetFile(file File)

	SetDir(dir Dir)
}

type memBlock struct {
	info *BlockInfo
	repo *MemRepo
}

func (block *memBlock) Parent() (FsNode, bool) {
	return block.repo.GetFile(block.info.Parent)
}

func (block *memBlock) Info() *BlockInfo {
	return block.info
}

func (block *memBlock) Repo() NodeRepo {
	return block.repo
}

type memFile struct {
	info *FileInfo
	repo *MemRepo

	blocks []string
}

func (file *memFile) Parent() (FsNode, bool) {
	return file.repo.GetDir(file.info.Parent)
}

func (file *memFile) Info() *FileInfo {
	return file.info
}

func (file *memFile) Name() string {
	return file.info.Name
}

func (file *memFile) Repo() NodeRepo {
	return file.repo
}

func (file *memFile) Blocks() []Block {
	blocks := []Block{}
	for _, strong := range file.blocks {
		block, has := file.repo.GetBlock(strong)
		blocks = append(blocks, block)
	}
	return blocks
}

type memDir struct {
	info *DirInfo
	repo *MemRepo

	subdirs []string
	files   []string
}

func (dir *memDir) Parent() (FsNode, bool) {
	return dir.repo.GetDir(dir.info.Parent)
}

func (dir *memDir) Info() *DirInfo {
	return dir.info
}

func (dir *memDir) Name() string {
	return dir.info.Name
}

func (dir *memDir) Repo() NodeRepo {
	return dir.repo
}

func (dir *memDir) SubDirs() []Dir {
	subdirs := []Dir{}
	for _, strong := range dir.subdirs {
		subdir, has := dir.repo.GetDir(strong)
		if has {
			subdirs = append(subdirs, subdir)
		}
	}
	return subdirs
}

func (dir *memDir) Files() []File {
	files := []File{}
	for _, strong := range dir.files {
		file, has := dir.repo.GetFile(strong)
		if has {
			files = append(files, file)
		}
	}
	return files
}

type MemRepo struct {
	blocks     map[string]*memBlock
	files      map[string]*memFile
	dirs       map[string]*memDir
	weakBlocks map[int]*memBlock
	root       FsNode
}

func NewMemRepo() *MemRepo {
	return &MemRepo{
		blocks: make(map[string]*memBlock),
		files:  make(map[string]*memFile),
		dirs:   make(map[string]*memDir)}
}

func (repo *MemRepo) Root() FsNode { return repo.root }

func (repo *MemRepo) CreateBlock(info *BlockInfo) Block {
	return &memBlock{info: info, repo: repo}
}

func (repo *MemRepo) CreateFile(info *FileInfo) File {
	return &memFile{info: info, repo: repo}
}

func (repo *MemRepo) NewDirID() string {
	return fmt.Sprintf("_%d", len(repo.dirs))
}

func (repo *MemRepo) CreateDir(info *DirInfo) Dir {
	dir := &memDir{info: info, repo: repo}
}

func (repo *MemRepo) GetWeakBlock(weak int) (block Block, has bool) {
	block, has = repo.weakBlocks[weak]
	return block, has
}

func (repo *MemRepo) GetBlock(strong string) (block Block, has bool) {
	block, has = repo.blocks[strong]
	return block, has
}

func (repo *MemRepo) GetFile(strong string) (file File, has bool) {
	file, has = repo.files[strong]
	return file, has
}

func (repo *MemRepo) GetDir(strong string) (dir Dir, has bool) {
	dir, has = repo.dirs[strong]
	return dir, has
}

func (repo *MemRepo) SetBlock(block Block) {
	mblock := repo.CreateBlock(block.Info()).(*memBlock)
	repo.blocks[block.Info().Strong] = mblock
	repo.weakBlocks[block.Info().Weak] = mblock
}

func (repo *MemRepo) SetFile(file File) {
	repo.files[file.Info().Strong] = repo.CreateFile(file.Info()).(*memFile)
}

func (repo *MemRepo) SetDir(dir Dir) {
	dirInfo := dir.Info()
	if dirInfo.Strong == "" {
		dirInfo = &DirInfo{
			Name:   dirInfo.Name,
			Mode:   dirInfo.Mode,
			Strong: fmt.Sprintf("_%d", len(repo.dirs)),
			Parent: dirInfo.Parent}
	}
	repo.dirs[dir.Info().Strong] = repo.CreateDir(dirInfo).(*memDir)
}
