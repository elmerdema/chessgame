const CopyWebpackPlugin = require("copy-webpack-plugin");
const path = require('path');

module.exports = {
  entry: "./bootstrap.js",
  output: {
    path: path.resolve(__dirname, "dist"),
    filename: "bootstrap.js",
  },
  mode: "development",
  plugins: [
    new CopyWebpackPlugin(['index.html'])
  ],
  experiments: {
    asyncWebAssembly: true, // Enable async WebAssembly support
    // or syncWebAssembly: true, // If you prefer synchronous WebAssembly (deprecated)
  },
  module: {
    rules: [
      {
        test: /\.wasm$/,
        type: "webassembly/async", // Ensure Webpack treats .wasm files correctly
      },
    ],
  },
};
