#include "llama_data_source.h"
#include <algorithm>
#include <cstring>
#include <stdexcept>
#include <sys/stat.h>

// File-based data source implementation
llama_file_source::llama_file_source(const char *filename)
    : fp(nullptr), file_size(0) {
    fp = fopen(filename, "rb");
    if (!fp) {
        throw std::runtime_error("Failed to open file");
    }

    // Get file size
    struct stat st;
    if (stat(filename, &st) == 0) {
        file_size = st.st_size;
    } else {
        fseek(fp, 0, SEEK_END);
        file_size = ftell(fp);
        fseek(fp, 0, SEEK_SET);
    }
}

llama_file_source::~llama_file_source() {
    if (fp) {
        fclose(fp);
    }
}

size_t llama_file_source::read(void *buffer, size_t size) {
    return fread(buffer, 1, size, fp);
}

void llama_file_source::seek(size_t offset, int whence) {
    fseek(fp, offset, whence);
}

size_t llama_file_source::tell() const { return ftell(fp); }

size_t llama_file_source::size() const { return file_size; }

bool llama_file_source::eof() const { return feof(fp) != 0; }

// Memory buffer data source implementation
llama_memory_source::llama_memory_source(const void *buffer, size_t buffer_size)
    : data(static_cast<const uint8_t *>(buffer)), data_size(buffer_size),
      current_pos(0) {}

size_t llama_memory_source::read(void *buffer, size_t size) {
    size_t bytes_to_read = std::min(size, data_size - current_pos);
    if (bytes_to_read > 0) {
        memcpy(buffer, data + current_pos, bytes_to_read);
        current_pos += bytes_to_read;
    }
    return bytes_to_read;
}

void llama_memory_source::seek(size_t offset, int whence) {
    switch (whence) {
    case SEEK_SET:
        current_pos = std::min(offset, data_size);
        break;
    case SEEK_CUR:
        current_pos = std::min(current_pos + offset, data_size);
        break;
    case SEEK_END:
        current_pos = data_size > offset ? data_size - offset : 0;
        break;
    }
}

size_t llama_memory_source::tell() const { return current_pos; }

size_t llama_memory_source::size() const { return data_size; }

bool llama_memory_source::eof() const { return current_pos >= data_size; }
