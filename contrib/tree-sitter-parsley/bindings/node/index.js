const root = require("path").join(__dirname, "..", "..");

module.exports =
  typeof process.versions.bun === "string"
    ? require(`${root}/build/Release/tree_sitter_parsley_binding.node`)
    : require("node-gyp-build")(root);

try {
  module.exports.nodeTypeInfo = require("../../src/node-types.json");
} catch (_) {}
