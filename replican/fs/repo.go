package fs

import (
	"fmt"
)

type NodeRepo interface {
	Root() FsNode

	WeakBlock(weak int) (Block, bool)

	Block(strong string) (Block, bool)

	File(strong string) (File, bool)

	Dir(strong string) (Dir, bool)

	AddBlock(file File, blockInfo *BlockInfo) Block

	AddFile(dir Dir, fileInfo *FileInfo, blocksInfo []*BlockInfo) File

	AddDir(dir Dir, subdirInfo *DirInfo) Dir

	Close()

	IndexFilter() IndexFilter
}

type memBlock struct {
	info   *BlockInfo
	repo   *MemRepo
	parent File
}

func (block *memBlock) Parent() (FsNode, bool) {
	file, is := block.parent.(*memFile)
	return file, is
}

func (block *memBlock) Info() *BlockInfo {
	return block.info
}

func (block *memBlock) Repo() NodeRepo {
	return block.repo
}

type memFile struct {
	info   *FileInfo
	repo   *MemRepo
	parent Dir
	blocks []Block
}

func (file *memFile) Parent() (FsNode, bool) {
	dir, is := file.parent.(*memDir)
	return dir, is
}

func (file *memFile) Info() *FileInfo {
	return file.info
}

func (file *memFile) Name() string {
	return file.info.Name
}

func (file *memFile) Mode() uint32 {
	return file.info.Mode
}

func (file *memFile) Repo() NodeRepo {
	return file.repo
}

func (file *memFile) Blocks() []Block {
	return file.blocks
}

type memDir struct {
	info    *DirInfo
	repo    *MemRepo
	parent  Dir
	files   []File
	subdirs []Dir
}

func (dir *memDir) Parent() (FsNode, bool) {
	parentDir, is := dir.parent.(*memDir)
	return parentDir, is
}

func (dir *memDir) Info() *DirInfo {
	return dir.info
}

func (dir *memDir) Name() string {
	return dir.info.Name
}

func (dir *memDir) Mode() uint32 {
	return dir.info.Mode
}

func (dir *memDir) Repo() NodeRepo {
	return dir.repo
}

func (dir *memDir) Files() []File {
	return dir.files
}

func (dir *memDir) SubDirs() []Dir {
	return dir.subdirs
}

func (dir *memDir) UpdateStrong() string {
	newStrong := CalcStrong(dir)
	if newStrong != dir.info.Strong {
		dir.repo.dirs[dir.info.Strong] = nil, false
		dir.repo.dirs[newStrong] = dir
		dir.info.Strong = newStrong
	}
	return newStrong
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
		blocks:     make(map[string]*memBlock),
		files:      make(map[string]*memFile),
		dirs:       make(map[string]*memDir),
		weakBlocks: make(map[int]*memBlock)}
}

func (repo *MemRepo) Root() FsNode { return repo.root }

func (repo *MemRepo) WeakBlock(weak int) (block Block, has bool) {
	block, has = repo.weakBlocks[weak]
	return block, has
}

func (repo *MemRepo) Block(strong string) (block Block, has bool) {
	block, has = repo.blocks[strong]
	return block, has
}

func (repo *MemRepo) File(strong string) (file File, has bool) {
	file, has = repo.files[strong]
	return file, has
}

func (repo *MemRepo) Dir(strong string) (dir Dir, has bool) {
	dir, has = repo.dirs[strong]
	return dir, has
}

func (repo *MemRepo) AddBlock(file File, info *BlockInfo) Block {
	block := &memBlock{repo: repo, info: info, parent: file}
	repo.blocks[info.Strong] = block
	repo.weakBlocks[info.Weak] = block
	mfile := file.(*memFile)
	mfile.blocks = append(mfile.blocks, block)
	return block
}

func (repo *MemRepo) AddFile(dir Dir, fileInfo *FileInfo, blocksInfo []*BlockInfo) File {
	file := &memFile{repo: repo, info: fileInfo, parent: dir}
	repo.files[fileInfo.Strong] = file
	for _, blockInfo := range blocksInfo {
		repo.AddBlock(file, blockInfo)
	}
	if mdir, is := dir.(*memDir); is {
		mdir.files = append(mdir.files, file)
	} else {
		repo.root = file
	}
	return file
}

func (repo *MemRepo) AddDir(dir Dir, info *DirInfo) Dir {
	if info.Strong == "" {
		info.Strong = fmt.Sprintf("tmp%d", len(repo.dirs))
	}
	subdir := &memDir{repo: repo, info: info, parent: dir}
	repo.dirs[info.Strong] = subdir
	if mdir, is := dir.(*memDir); is {
		mdir.subdirs = append(mdir.subdirs, subdir)
	} else {
		repo.root = subdir
	}
	return subdir
}

func (repo *MemRepo) Close() {
}

func (repo *MemRepo) IndexFilter() IndexFilter {
	return AlwaysMatch
}
