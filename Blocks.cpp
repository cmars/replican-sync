
#include "Blocks.hpp"

#include <boost/filesystem.hpp>
#include <boost/system/error_code.hpp>
#include <boost/pointer_cast.hpp>

#include <iostream>
#include <fstream>
#include <sstream>
#include <iomanip>

#include <stack>

#include <openssl/sha.h>

using namespace replican;
namespace fs = boost::filesystem;
namespace sys = boost::system;

static NodePtr nullNode;
static FilePtr nullFile;

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

const std::string& Node::get_strong() { return strong; }

void Node::set_digest(const unsigned char* raw, int len) {
    std::stringstream sout;
    sout << std::hex << std::setfill('0') << std::setw(2);
    for (int i = 0; i < len; i++) {
        sout << (int)raw[i];
    }
    strong = sout.str();
}

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
        boost::shared_ptr<FsNode> fsNode = boost::dynamic_pointer_cast<FsNode>(current);
        parts.push_back(fsNode->get_name());
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

const std::string& Dir::get_strong() {
    if (!strong.empty()) {
        return strong;
    }
    
    std::stringstream sout;
    sout << *this;
    std::string repr = sout.str();
    
    SHA_CTX context;
    unsigned char digest[SHA_DIGEST_LENGTH];
    SHA1_Init(&context);
    SHA1_Update(&context, (unsigned char*)repr.c_str(), repr.size());
    SHA1_Final(digest, &context);
    
    set_digest(digest, SHA_DIGEST_LENGTH);
    return strong;
}

std::ostream& std::operator<<(std::ostream& out, const Dir& dir) {
    for (NodePtrVector::const_iterator i = dir.get_children().begin(); i < dir.get_children().end(); i++) {
//        NodePtr node = *i;
//        boost::shared_ptr<FsNode> child = boost::dynamic_pointer_cast<FsNode>(node);
//        out << child->get_name() << "\t" << child->get_strong() << std::endl;
        
        FsNode* child = (FsNode*)i->get();
        out << child->get_name() << "\t" << child->get_strong() << std::endl;
    }
}

DirPtr replican::index_dir(const fs::path& root_path) {
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
                const fs::path& entry = *iter;
//                std::cout << entry << std::endl;
                try {
                    if (fs::is_directory(entry)) {
                        dirstack.push(entry);
                    }
                    else if (fs::is_regular_file(entry)) {
                        FilePtr file = index_file(entry);
                        if (file.get()) {
                            current->add_child(file);
                        }
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

FilePtr replican::index_file(const fs::path& file) {
    char buf[BLOCKSIZE];
    
    SHA_CTX block_context;
    SHA_CTX file_context;
    unsigned char digest[SHA_DIGEST_LENGTH];
    
    FilePtr f(new File(file.filename().string()));
    std::ifstream ifs(file.c_str(), std::ios::in | std::ios::binary);
    
    SHA1_Init(&file_context);
    
    while (ifs.good()) {
        SHA1_Init(&block_context);
        
        BlockPtr b(new Block(ifs.tellg()));
        ifs.read(buf, BLOCKSIZE);
        
        if (ifs.bad()) {
            break;
        }
        
        int rd_gcount = ifs.gcount();
        SHA1_Update(&block_context, (unsigned char*)buf, rd_gcount);
        SHA1_Update(&file_context, (unsigned char*)buf, rd_gcount);
        
        SHA1_Final(digest, &block_context);
        
        b->set_digest(digest, SHA_DIGEST_LENGTH);
        f->add_child(b);
    }
    
    if (ifs.bad()) {
        ifs.close();
        return nullFile;
    }
    
    SHA1_Final(digest, &file_context);
    f->set_digest(digest, SHA_DIGEST_LENGTH);
    
    ifs.close();
    return f;
}


