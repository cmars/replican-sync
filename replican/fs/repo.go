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
	
	Blocks(file string) []Block
	
	Files(dir string) []File
	
	SubDirs(dir string) []Dir
	
	SetBlock(block *BlockInfo) Block

	SetFile(file *FileInfo) File

	SetDir(dir *DirInfo) Dir
	
	Remove(strong string)
}

type memBlock struct {
	info *BlockInfo
	repo *MemRepo
}

func (block *memBlock) Parent() (FsNode, bool) {
	return block.repo.File(block.info.Parent)
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
}

func (file *memFile) Parent() (FsNode, bool) {
	return file.repo.Dir(file.info.Parent)
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
	return file.repo.Blocks(file.info.Strong)
}

type memDir struct {
	info *DirInfo
	repo *MemRepo
}

func (dir *memDir) Parent() (FsNode, bool) {
	return dir.repo.Dir(dir.info.Parent)
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

func (dir *memDir) Files() []File {
	return dir.repo.Files(dir.info.Strong)
}

func (dir *memDir) SubDirs() []Dir {
	return dir.repo.SubDirs(dir.info.Strong)
}

type MemRepo struct {
	blocks     map[string]*memBlock
	files      map[string]*memFile
	dirs       map[string]*memDir
	weakBlocks map[int]*memBlock
	children   map[string][]string
	root       FsNode
}

func NewMemRepo() *MemRepo {
	return &MemRepo{
		blocks: make(map[string]*memBlock),
		files:  make(map[string]*memFile),
		dirs:   make(map[string]*memDir),
		weakBlocks: make(map[int]*memBlock),
		children: make(map[string][]string) }
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

func (repo *MemRepo) Blocks(parent string) []Block {
	result := []Block{}
	if children, hasChildren := repo.children[parent]; hasChildren {
		for _, child := range children {
			if block, has := repo.blocks[child]; has {
				result = append(result, block)
			}
		}
	}
	return result
}

func (repo *MemRepo) Files(parent string) []File {
	result := []File{}
	if children, hasChildren := repo.children[parent]; hasChildren {
		for _, child := range children {
			if file, has := repo.files[child]; has {
				result = append(result, file)
			}
		}
	}
	return result
}

func (repo *MemRepo) SubDirs(parent string) []Dir {
	result := []Dir{}
	if children, hasChildren := repo.children[parent]; hasChildren {
		for _, child := range children {
			if dir, has := repo.dirs[child]; has {
				result = append(result, dir)
			}
		}
	}
	return result
}

func (repo *MemRepo) SetBlock(block *BlockInfo) Block {
	mblock := &memBlock{ repo: repo, info: block }
	repo.blocks[mblock.info.Strong] = mblock
	repo.weakBlocks[mblock.info.Weak] = mblock
	
	if mblock.info.Parent != "" {
		repo.addChild(mblock.info.Parent, mblock.info.Strong)
	}
	return mblock
}

func (repo *MemRepo) SetFile(file *FileInfo) File {
	mfile := &memFile{ repo: repo, info: file }
	repo.files[file.Strong] = mfile
	
	if mfile.info.Parent != "" {
		repo.addChild(mfile.info.Parent, mfile.info.Strong)
	}
	return mfile
}

func (repo *MemRepo) SetDir(dir *DirInfo) Dir {
	if dir.Strong == "" {
		dir.Strong = fmt.Sprintf("tmp%d", len(repo.dirs))
	}
	
	mdir := &memDir{ repo: repo, info: dir }
	repo.dirs[dir.Strong] = mdir
	
	if mdir.info.Parent != "" {
		repo.addChild(mdir.info.Parent, mdir.info.Strong)
	}
	return mdir
}

func (repo *MemRepo) addChild(parent string, child string) {
	children, has := repo.children[parent]
	if !has {
		children = []string{}
	}
	
	repo.children[parent] = append(children, child)
}

func (repo *MemRepo) Remove(strong string) {
	repo.blocks[strong] = nil, false
	repo.files[strong] = nil, false
	repo.dirs[strong] = nil, false
	repo.children[strong] = nil, false
}
