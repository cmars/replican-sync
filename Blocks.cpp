
#include "Blocks.hpp"
#include <boost/filesystem.hpp>
#include <boost/system/error_code.hpp>
#include <boost/pointer_cast.hpp>
#include <iostream>
#include <stack>

using namespace replican;
namespace fs = boost::filesystem;
namespace sys = boost::system;

static NodePtr nullNode;

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

Block::Block(int _offset): Node(), offset(_offset) {}

Block::~Block() {}

FsNode::FsNode(const std::string& _name): Node(), name(_name) {}

FsNode::~FsNode() {}

File::File(const std::string& _name): FsNode(_name) {}

File::~File() {}

fs::path FsNode::get_path() {
    std::vector<std::string> parts;
    parts.push_back(name);
    
    for (NodePtr current = get_parent(); current.get(); current = current->get_parent()) {
        FsNode& fsNode = static_cast<FsNode&>(*current);
        parts.push_back(fsNode.get_name());
    }
    
    fs::path result;
    for (std::vector<std::string>::reverse_iterator i = parts.rbegin();
            i < parts.rend(); ++i) {
        result /= *i;
    }
    return result;
}

Dir::Dir(const std::string& _name): FsNode(_name) {}

Dir::~Dir() {}

DirPtr replican::index_dir(fs::path& root_path) {
    std::stack<fs::path> dirstack;
    dirstack.push(root_path);
    
    DirPtr root(new Dir(root_path.filename().string()));
    NodePtr parent;
    NodePtr current(root);
    
    while (!dirstack.empty()) {
        fs::path current_path = dirstack.top();
        dirstack.pop();
        
        if (parent.get()) {
            current.reset(new Dir(current_path.filename().string()));
            parent->add_child(current);
        }
        
        try {
            fs::directory_iterator end, iter, start(current_path);
            for (iter = start; iter != end; ++iter) {
                const fs::path entry = *iter;
//                std::cout << entry << std::endl;
                try {
                    if (fs::is_directory(entry)) {
                        dirstack.push(entry);
                    }
                    else if (fs::is_regular_file(entry)) {
                        NodePtr file(new File(entry.string()));
                        current->add_child(file);
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
        
        parent = current;
    }
    
    return root;
}

FilePtr index_file(fs::path& file) {
    char buf[BLOCKSIZE];
    FilePtr f(new File(file.filename()));
    /*
    std::ifstream ifs(file.string(), ios::in | ios::binary);
    
    while (
    ifs.read(buf, BLOCKSIZE);
    */
}


