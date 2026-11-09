import * as esbuild from "esbuild";
import { fileURLToPath } from "node:url";
import path from "node:path";

const root = path.dirname(fileURLToPath(import.meta.url));

await esbuild.build({
  absWorkingDir: root,
  entryPoints: ["src/index.tsx"],
  outfile: path.join(root, "..", "ui.js"),
  bundle: true,
  format: "iife",
  platform: "browser",
  target: ["es2020"],
  jsx: "transform",
  jsxFactory: "React.createElement",
  jsxFragment: "React.Fragment",
  minify: true,
  legalComments: "none",
  plugins: [
    {
      name: "window-globals",
      setup(build) {
        build.onResolve({ filter: /^(react|antd)$/ }, (args) => ({
          path: args.path,
          namespace: "window-globals",
        }));
        build.onLoad({ filter: /.*/, namespace: "window-globals" }, (args) => {
          if (args.path === "react") {
            return { contents: "module.exports = window.React;", loader: "js" };
          }
          return { contents: "module.exports = window.antd;", loader: "js" };
        });
      },
    },
  ],
});
