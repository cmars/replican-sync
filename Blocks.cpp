
#include "Blocks.hpp"
#include "boost/filesystem.hpp"
#include "boost/system/error_code.hpp"
#include <iostream>
#include <stack>

using namespace replican;
namespace fs = boost::filesystem;
namespace sys = boost::system;

static DirPtr nullDir;

WeakChecksum::WeakChecksum(): a(0), b(0) {}

WeakChecksum::WeakChecksum(int _a, int _b): a(_a), b(_b) {}

WeakChecksum::~WeakChecksum() {}

void WeakChecksum::update(int len, char* buf) {
    for (int i = 0; i < len; i++) {
        a += buf[i];
        b += (len - i) * buf[i];
    }
}

Node::~Node() {}

Block::Block(const FilePtr _file, int _offset): file(_file), offset(_offset) {}

Block::~Block() {}

FsNode::FsNode(const std::string& _name): dir(DirPtr(nullDir.get())), name(_name) {}

FsNode::FsNode(const DirPtr _dir, const std::string& _name): dir(_dir), name(_name) {}

FsNode::~FsNode() {}

File::File(const DirPtr _dir, const std::string& _name): FsNode(_dir, _name) {}

File::~File() {}

Dir::Dir(const std::string& _name): FsNode(_name) {}

Dir::Dir(const DirPtr _dir, const std::string& _name): FsNode(_dir, _name) {}

Dir::~Dir() {}

DirPtr replican::index(fs::path& root_path) {
    std::stack<fs::path> dirstack;
    dirstack.push(root_path);
    
    while (!dirstack.empty()) {
        fs::path current = dirstack.top();
        dirstack.pop();
        
        try {
            fs::directory_iterator end, iter, start(current);
            for (iter = start; iter != end; ++iter) {
                const fs::path entry = *iter;
                std::cout << entry << std::endl;
                try {
                    if (fs::is_directory(entry)) {
                        dirstack.push(entry);
                    }
                }
                catch (const fs::filesystem_error& err) {
                    std::cerr << "Error getting status for " << entry << ": " << err.what() << std::endl;
                }
            }
        }
        catch (const fs::filesystem_error& err) {
            std::cerr << "Error reading directory " << current << ": " << err.what() << std::endl;
        }
    }
    
    return nullDir;
}


