
local M = {}

local io=require"io"
local table=require"table"

local crypto=require"crypto"
local lfs=require"lfs"

local dir=require"pl.dir"
local plpath=require"pl.path"
local class=require"pl.class"

M.BLOCKSIZE=8192

function M.bintohex(s)
  return (s:gsub('(.)', function(c)
    return string.format('%02x', string.byte(c))
  end))
end 

-- Start a weak checksum on a block of data
-- Good for a rolling checksum
function M.start_cksum(data)
    local a = 0
    local b = 0
    local l = data:len()
    local x
    
    for i = 1, l do
        x = data:byte(i)
        a = a + x
        b = b + (l - i) * x
    end
    
    return a, b
end

-- Complete weak checksum on a smallish block
function M.weak_cksum(data)
    local a, b = M.start_cksum(data)
    return (b * 65536) + a
end

-- Roll checksum byte-by-byte
function M.roll_cksum(removed_byte, added_byte, a, b)
    a = a - (removed_byte - added_byte)
    b = b - ((removed_byte * M.BLOCKSIZE) - a)
    return a, b
end

function M.strong_cksum(data)
    return crypto.evp.digest("sha1", data)
end

M.FileIndex = class()

function M.FileIndex:_init(path)
    self.path = path
    self.strong = nil
    self.blocks = {}
end

function M.get_file_index(path)
    local index = M.FileIndex(path)
    local f = io.open(path, "r")
    local block_num = 1
    local buf
    local hash = crypto.evp.new("sha1")
    
    while true do
        buf = f:read(M.BLOCKSIZE)
        if not buf then
            break
        end
        
        index.blocks[block_num] = {weak=M.weak_cksum(buf), strong=M.strong_cksum(buf)}
        
        hash:update(buf)
        
        block_num = block_num + 1
    end
    
    io.close(f)
    index.strong = hash:digest()
    
    return index
end

function M.get_file_block(path, block_nums)
    local result = {}
    local f = io.open(path, "r")
    
    for block_num in pairs(block_nums) do
        f:seek(block_num * M.BLOCKSIZE)
        result[block_num] = f:read(M.BLOCKSIZE)
    end
    
    return result
end

M.DirIndex = class()

function M.DirIndex:_init(path)
    self.path = path
    self.dirs = {}
    self.files = {}
    self.strong = nil
end

function M.get_dir_index(path)
    local spread = {}
    
    for root, subdirs, files in dir.walk(path) do
        table.insert(spread, M.DirIndex(root))
        
        for i, f in pairs(files) do
            local fpath = plpath.join(root, f)
            table.insert(spread, M.get_file_index(fpath))
        end
    
    end
    
    return spread
end

return M

