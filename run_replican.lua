
require"luarocks.loader"
local lanes=require"lanes"
local cuteadmin=require"cuteadmin"
local treekeeper=require"treekeeper"

local linda = lanes.linda()

tk_h = lanes.gen("*", treekeeper.start)(linda, "/var/tmp")

ca_h = lanes.gen("*", cuteadmin.start)(linda, 9009)

ca_h:join()
tk_h:join()

