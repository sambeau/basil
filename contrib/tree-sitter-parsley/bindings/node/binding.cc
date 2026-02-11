#include <napi.h>

typedef struct TSLanguage TSLanguage;

extern "C" TSLanguage *tree_sitter_parsley();

Napi::Object Init(Napi::Env env, Napi::Object exports) {
  exports["name"] = Napi::String::New(env, "parsley");
  auto language = Napi::External<TSLanguage>::New(env, tree_sitter_parsley());
  exports["language"] = language;
  return exports;
}

NODE_API_MODULE(tree_sitter_parsley_binding, Init)