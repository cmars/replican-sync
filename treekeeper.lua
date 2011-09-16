
local M = {}

local io=require"io"
local class=require"pl.class"
local lanes=require"lanes"
local blocks=require"blocks"

M.TreeKeeper = class()

function M.TreeKeeper:_init(root)
    self.root = root
end

function M.TreeKeeper:run(linda)
    print("treekeeper starting...\n")
    self.linda = linda
    local msg
    local key
    
    lanes.timer(linda, "reindex", 3, 300)
    
    while true do
        local resp = nil
        msg, key = linda:receive("get-tree", "get-block", "reindex")
        
        if key == "get-tree" then
            resp = self:get_tree(msg)
        elseif key == "get-block" then
            resp = self:get_block(msg)
        elseif key == "reindex" then
            io.write("reindexing...")
            self.tree = blocks.get_dir_index(self.root)
            io.write("reindexing complete\n")
        elseif key == "shutdown" then
            return
        end
        
        if resp then
            linda:send(key .. "-resp-" .. msg.seq, resp)
        end
    end
end

function M.TreeKeeper:get_tree(msg)
    return self.tree
end

function M.TreeKeeper:get_block(msg)
    local path = self:get_local_path(msg.path)
    return blocks.get_file_block(path, msg.block_nums)
end

return M

