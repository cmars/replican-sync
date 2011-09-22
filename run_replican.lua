
require"luarocks.loader"
require"lanes"

local linda = lanes.linda()

local function run_treekeeper(root)
    print"hi"
    local treekeeper=require"treekeeper"
    print"hi"
    local tk = treekeeper.TreeKeeper(root)
    tk:run(linda)
end

local function run_server(port)
    local server=require"server"
    local s = server.Server(port)
    s:run(linda)
end

tk_h = lanes.gen("*", run_treekeeper)("/home/casey/src-lua/replican/testroot")

s_h = lanes.gen("*", run_server)(9009)

print(s_h[1], tk_h[1])

tk_h:join()
s_h:join()

