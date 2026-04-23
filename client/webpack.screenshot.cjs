/**
 * Minimal webpack config for screenshot harness.
 * Skips Module Federation.
 */
const path = require('path')
const HtmlWebpackPlugin = require('html-webpack-plugin')

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
          loader: 'ts-loader',
          options: {
            configFile: path.resolve(__dirname, 'tsconfig.json'),
            transpileOnly: true,
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
                // Inline Tailwind config — production tailwind.config.js uses ESM
                // imports that don't resolve in CJS postcss-loader context
                plugins: [
                  ['tailwindcss', {
                    content: [
                      'src/**/*.{ts,tsx}',
                      'node_modules/@tumaet/prompt-ui-components/dist/**/*.{js,ts,tsx}',
                    ],
                  }],
                  'autoprefixer',
                ],
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
  },
  resolve: {
    extensions: ['.ts', '.tsx', '.js', '.mjs', '.jsx'],
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: 'public/template.html',
    }),
  ],
  cache: { type: 'filesystem' },
}
