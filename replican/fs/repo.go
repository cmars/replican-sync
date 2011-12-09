package fs

import (
	"fmt"
	"os"
	"sort"
	"strconv"
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
	blocks     map[string]*BlockInfo
	files      map[string]*FileInfo
	dirs       map[string]*DirInfo
	weakBlocks map[int]*BlockInfo
	children   map[string]map[string]string
	root       FsNode
}

func NewMemRepo() *MemRepo {
	return &MemRepo{
		blocks: make(map[string]*BlockInfo),
		files:  make(map[string]*FileInfo),
		dirs:   make(map[string]*DirInfo),
		weakBlocks: make(map[int]*BlockInfo),
		children: make(map[string]map[string]string) }
}

func (repo *MemRepo) Root() FsNode { return repo.root }

func (repo *MemRepo) WeakBlock(weak int) (block Block, has bool) {
	blockInfo, has := repo.weakBlocks[weak]
	return &memBlock{ repo: repo, info: blockInfo }, has
}

func (repo *MemRepo) Block(strong string) (block Block, has bool) {
	blockInfo, has := repo.blocks[strong]
	return &memBlock{ repo: repo, info: blockInfo }, has
}

func (repo *MemRepo) File(strong string) (file File, has bool) {
	fileInfo, has := repo.files[strong]
	return &memFile{ repo: repo, info: fileInfo }, has
}

func (repo *MemRepo) Dir(strong string) (dir Dir, has bool) {
	dirInfo, has := repo.dirs[strong]
	return &memDir{ repo: repo, info: dirInfo }, has
}

func (repo *MemRepo) Blocks(parent string) []Block {
	result := &Blocks{}
	if children, hasChildren := repo.children[parent]; hasChildren {
		for name, child := range children {
			if blockInfo, has := repo.blocks[child]; has {
				newInfo := *blockInfo
				newInfo.Parent = parent
				var err os.Error
				if newInfo.Position, err = strconv.Atoi(name); err == nil {
					result.Contents = append(result.Contents, 
						&memBlock{ repo: repo, info: &newInfo })
				}
			}
		}
	}
	sort.Sort(result)
	return result.Contents
}

func (repo *MemRepo) Files(parent string) []File {
	result := &Files{}
	if children, hasChildren := repo.children[parent]; hasChildren {
		for name, child := range children {
			if fileInfo, has := repo.files[child]; has {
				var newInfo FileInfo
				newInfo = *fileInfo
				newInfo.Parent = parent
				newInfo.Name = name
				result.Contents = append(result.Contents, &memFile{ repo: repo, info: &newInfo })
			}
		}
	}
	sort.Sort(result)
	return result.Contents
}

func (repo *MemRepo) SubDirs(parent string) []Dir {
	result := &Dirs{}
	if children, hasChildren := repo.children[parent]; hasChildren {
		for name, child := range children {
			if dirInfo, has := repo.dirs[child]; has {
				newInfo := *dirInfo
				newInfo.Parent = parent
				newInfo.Name = name
				result.Contents = append(result.Contents, &memDir{ repo: repo, info: &newInfo })
			}
		}
	}
	sort.Sort(result)
	return result.Contents
}

func (repo *MemRepo) SetBlock(block *BlockInfo) Block {
	mblock := &memBlock{ repo: repo, info: block }
	repo.blocks[block.Strong] = block
	repo.weakBlocks[block.Weak] = block
	
	if block.Parent != "" {
		repo.addChild(block.Parent, block.Strong, fmt.Sprintf("%d", block.Position))
	}
	return mblock
}

func (repo *MemRepo) SetFile(file *FileInfo) File {
	mfile := &memFile{ repo: repo, info: file }
	repo.files[file.Strong] = file
	
	if file.Parent != "" {
		repo.addChild(file.Parent, file.Strong, file.Name)
	}
	return mfile
}

func (repo *MemRepo) SetDir(dir *DirInfo) Dir {
	if dir.Strong == "" {
		dir.Strong = fmt.Sprintf("tmp%d", len(repo.dirs))
	}
	
	mdir := &memDir{ repo: repo, info: dir }
	repo.dirs[dir.Strong] = dir
	
	if dir.Parent != "" {
		repo.addChild(dir.Parent, dir.Strong, dir.Name)
	}
	return mdir
}

func (repo *MemRepo) addChild(parent string, child string, name string) {
	children, has := repo.children[parent]
	if !has {
		children = make(map[string]string)
		repo.children[parent] = children
	}
	children[name] = child
}

func (repo *MemRepo) Remove(strong string) {
	repo.blocks[strong] = nil, false
	repo.files[strong] = nil, false
	repo.dirs[strong] = nil, false
	repo.children[strong] = nil, false
}
