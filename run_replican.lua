
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

local function run_cuteadmin(port)
    local cuteadmin=require"cuteadmin"
    local ca = cuteadmin.CuteAdmin(port)
    ca:run(linda)
end

tk_h = lanes.gen("*", run_treekeeper)("/home/casey/sketchbook")

ca_h = lanes.gen("*", run_cuteadmin)(9009)

print(ca_h[1], tk_h[1])

tk_h:join()
ca_h:join()

