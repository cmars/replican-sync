
local M = {}

local io=require"io"
local copas=require"copas"
local socket=require"socket"
local class=require"pl.class"
local blocks=require"blocks"
local json=require"cjson"
local lanes=require"lanes"

M.Server = class()

function M.Server:_init(port)
    self.port = port
    self.seq = 1
end

function M.Server:run(linda)
    io.write("server starting...\n")
    self.linda = linda
    local server = socket.bind("0.0.0.0", self.port)
    copas.addserver(server, function(socket)
        self:handler(socket)
    end)
    copas.loop()
end

function M.Server:handler(socket)
    local data = copas.receive(socket)
    local msg = json.decode(data)
    local resp
    
    msg.seq = self.seq
    self.seq = self.seq + 1
    
    local respkey = msg.command .. "-resp-" .. msg.seq
    
    io.write("got command: " .. msg.command .. "\n")
    
    self.linda:send(msg.command, msg)
    resp = self.linda:receive(0.001, respkey)
    while not resp do
        resp = self.linda:receive(0.001, respkey)
    end
    
    if resp then
        copas.send(socket, json.encode(resp))
    end
    
end

return M

