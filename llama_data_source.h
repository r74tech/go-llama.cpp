#ifndef LLAMA_DATA_SOURCE_H
#define LLAMA_DATA_SOURCE_H

#include <cstddef>
#include <cstdint>
#include <cstdio>
#include <memory>

// Abstract data source interface for reading model data
class llama_data_source {
  public:
    virtual ~llama_data_source() = default;
    virtual size_t read(void *buffer, size_t size) = 0;
    virtual void seek(size_t offset, int whence) = 0;
    virtual size_t tell() const = 0;
    virtual size_t size() const = 0;
    virtual bool eof() const = 0;
};

// File-based data source implementation
class llama_file_source : public llama_data_source {
  private:
    FILE *fp;
    size_t file_size;

  public:
    llama_file_source(const char *filename);
    ~llama_file_source();

    size_t read(void *buffer, size_t size) override;
    void seek(size_t offset, int whence) override;
    size_t tell() const override;
    size_t size() const override;
    bool eof() const override;
};

// Memory buffer data source implementation
class llama_memory_source : public llama_data_source {
  private:
    const uint8_t *data;
    size_t data_size;
    size_t current_pos;

  public:
    llama_memory_source(const void *buffer, size_t buffer_size);
    ~llama_memory_source() = default;

    size_t read(void *buffer, size_t size) override;
    void seek(size_t offset, int whence) override;
    size_t tell() const override;
    size_t size() const override;
    bool eof() const override;
};

#endif // LLAMA_DATA_SOURCE_H
