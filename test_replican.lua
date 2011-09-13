
require"luarocks.loader"
require"lunit"

local io=require"io"
local json=require"cjson"
local blocks=require"blocks"

local pairs=pairs

module("test_replican", lunit.testcase)

function test_file_index()
    local result = blocks.get_file_index("testroot/My Music/0 10k 30.mp4")
    assert_equal("5ab3e5d621402e5894429b5f595a1e2d7e1b3078", result.strong)
end

function test_dir_index()
    local result = blocks.get_dir_index("testroot")
    
    io.write(json.encode(result))
    --[[--
    for i, index in pairs(result) do
        io.write(index.path .. "\n")
    --]]--
end


