
local M = {}

local io=require"io"
local class=require"pl.class"
local lanes=require"lanes"
local socket=require"socket"
local json=require"cjson"
local blocks=require"blocks"

M.Merger = class()

function M.Merger:_init()
    self.seq = 0    
end

function M.Merger:run(linda)
    print("merger starting...\n")
    self.linda = linda
    local msg
    local key
    
    while true do
        local resp = nil
        msg, key = linda:receive("merge-tree")
        
        if key == "merge-tree" then
            self:merge_tree(msg)
        elseif key == "shutdown" then
            return
        end
    end
end

function M.Merger:merge_tree(msg)
    local client = socket.connect(msg.address, msg.port)
    client:send(json.encode({command="get-tree"}))
    
    local remote_resp = client:receive("*a")
    local remote_tree = json.decode(remote_resp)
    
    local local_tree = self:get_local_tree()
    
    local merge_plan = M.get_merge_plan(local_tree, remote_tree)
    for _, merge_command in ipairs(merge_plan)
        self:apply_command(merge_command)
    end
end

function M.Merger:get_local_tree()
    self.seq = self.seq + 1
    self.linda:send({command="get-tree", seq="merger" .. self.seq})
    local resp = self.linda:receive("get-tree-resp-merger" .. self.seq)
    return json.decode(resp)
end

return M

