
#include "Blocks.hpp"
#include <boost/filesystem.hpp>
#include <boost/system/error_code.hpp>
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

fs::path FsNode::get_path() {
    std::vector<std::string> parts;
    parts.push_back(name);
    
    for (NodePtr current = get_parent(); !is_null(current); current = replican::get_parent(current)) {
        switch(current.which()) {
        case FILE_PTR:
            parts.push_back(boost::get<FilePtr>(current)->get_name());
            break;
            
        case DIR_PTR:
            parts.push_back(boost::get<DirPtr>(current)->get_name());
            break;
        
        // TODO: handle this unlikely turn of events???
        }
    }
    
    fs::path result;
    for (std::vector<std::string>::reverse_iterator i = parts.rbegin();
            i < parts.rend(); ++i) {
        result /= *i;
    }
    return result;
}

Dir::Dir(const std::string& _name): FsNode(_name) {}

Dir::Dir(const DirPtr _dir, const std::string& _name): FsNode(_dir, _name) {}

Dir::~Dir() {}

DirPtr replican::index_dir(fs::path& root_path) {
    std::stack<fs::path> dirstack;
    dirstack.push(root_path);
    
    DirPtr root(new Dir(root_path.filename().string()));
    DirPtr parent;
    DirPtr current(root);
    
    while (!dirstack.empty()) {
        fs::path current_path = dirstack.top();
        dirstack.pop();
        
        if (parent.get()) {
            current.reset(new Dir(parent, current_path.filename().string()));
            parent->get_children().push_back(current);
        }
        
        try {
            fs::directory_iterator end, iter, start(current_path);
            for (iter = start; iter != end; ++iter) {
                const fs::path entry = *iter;
                std::cout << entry << std::endl;
                try {
                    if (fs::is_directory(entry)) {
                        dirstack.push(entry);
                    }
                    else if (fs::is_regular_file(entry)) {
                        FilePtr file(new File(current, entry.string()));
//                        file->index();
                        current->get_children().push_back(file);
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
    
    return nullDir;
}


