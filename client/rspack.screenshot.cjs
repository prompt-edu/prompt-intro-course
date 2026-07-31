/**
 * Minimal Rspack config for the screenshot harness.
 * Skips Module Federation.
 */
const path = require('node:path')
const rspack = require('@rspack/core')

module.exports = {
  target: 'web',
  mode: 'development',
  devtool: 'source-map',
  entry: './src/screenshotEntry.tsx',
  devServer: {
    static: { directory: path.join(__dirname, 'public') },
    compress: true,
    hot: true,
    historyApiFallback: true,
    port: 3006,
    open: false,
    proxy: [
      {
        context: ['/intro-course'],
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    ],
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: {
          loader: 'builtin:swc-loader',
          options: {
            jsc: {
              parser: { syntax: 'typescript', tsx: true },
              transform: { react: { runtime: 'automatic' } },
            },
          },
        },
        exclude: /node_modules/,
      },
      {
        test: /\.css$/i,
        use: [
          'style-loader',
          'css-loader',
          {
            loader: 'postcss-loader',
            options: {
              postcssOptions: {
                plugins: {
                  '@tailwindcss/postcss': {},
                },
              },
            },
          },
        ],
        exclude: /node_modules/,
      },
      {
        test: /\.css$/i,
        include: /node_modules/,
        use: ['style-loader', 'css-loader'],
      },
    ],
  },
  output: {
    filename: 'screenshot.js',
    path: path.resolve(__dirname, 'build-screenshot'),
    publicPath: '/',
    clean: true,
  },
  resolve: {
    extensions: ['.ts', '.tsx', '.js', '.mjs', '.jsx'],
  },
  plugins: [
    new rspack.HtmlRspackPlugin({
      template: 'public/template.html',
    }),
  ],
  cache: { type: 'persistent' },
}
