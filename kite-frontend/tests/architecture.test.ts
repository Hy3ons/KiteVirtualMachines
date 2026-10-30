import { readdirSync, readFileSync } from 'node:fs';
import { dirname, relative, resolve } from 'node:path';
import ts from 'typescript';
import { describe, expect, it } from 'vitest';

const sourceRoot = resolve(import.meta.dirname, '../src');
type ImportEdge = { readonly from: string; readonly to: string; readonly typeOnly: boolean };

// forbiddenBoundary describes runtime dependencies that would invert the frontend layers.
// Paths are normalized relative to src; erased type imports may reference shared models.
function forbiddenBoundary({ from, to, typeOnly }: ImportEdge): boolean {
  if (typeOnly) return false;
  const contract = /^(?:api\/contracts(?:\/|$)|api\/(?:contracts|configContracts)$)/;
  if (contract.test(from)) return to !== 'zod' && !contract.test(to);
  if (from.startsWith('domain/')) return !to.startsWith('domain/');
  if (from.startsWith('api/')) return /^(?:server|app|screens|store|components)\//.test(to) || (from !== 'api/pageData' && to === 'api/pageData');
  if (/^(?:screens|components|hooks|store)\//.test(from)) return /^(?:server|app)\//.test(to);
  return false;
}

// importEdges reads static and literal dynamic imports without executing product modules.
// filename selects the importing file; source is either repository code or a negative fixture.
function importEdges(filename: string, source: string): readonly ImportEdge[] {
  const from = relative(sourceRoot, filename).replace(/\.(?:tsx?|d\.ts)$/, '');
  const edges: ImportEdge[] = [];
  const add = (specifier: string, typeOnly: boolean) => {
    const target = specifier.startsWith('.') ? relative(sourceRoot, resolve(dirname(filename), specifier)) : specifier.replace(/^@\//, '');
    edges.push({ from, to: target.replace(/\.(?:tsx?|d\.ts)$/, ''), typeOnly });
  };
  const visit = (node: ts.Node) => {
    if ((ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) && node.moduleSpecifier && ts.isStringLiteral(node.moduleSpecifier)) {
      const typeOnly = ts.isImportDeclaration(node) ? node.importClause?.isTypeOnly === true : node.isTypeOnly;
      add(node.moduleSpecifier.text, typeOnly);
    }
    if (ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword) {
      const argument = node.arguments.at(0);
      if (argument && ts.isStringLiteral(argument)) add(argument.text, false);
    }
    ts.forEachChild(node, visit);
  };
  visit(ts.createSourceFile(filename, source, ts.ScriptTarget.Latest, true));
  return edges;
}

describe('frontend dependency direction', () => {
  it('keeps contracts/domain independent and browser features below the server layer', () => {
    // Given every production TypeScript source, including dynamically loaded debug code.
    const files = readdirSync(sourceRoot, { recursive: true, encoding: 'utf8' }).filter((path) => /\.tsx?$/.test(path));
    // When imports are checked against the layer boundaries.
    const violations = files.flatMap((path) => {
      const filename = resolve(sourceRoot, path);
      return importEdges(filename, readFileSync(filename, 'utf8')).filter(forbiddenBoundary);
    });
    // Then no product module points into a higher layer.
    expect(violations).toEqual([]);
  });

  it('detects a runtime dependency from a domain policy into React infrastructure', () => {
    // Given an inverted dependency written as a literal dynamic import.
    const source = "export const load = () => import('../store/useAuthStore');";
    // When the same checker examines it.
    const violations = importEdges(resolve(sourceRoot, 'domain/invalid.ts'), source).filter(forbiddenBoundary);
    // Then the violation is found instead of silently passing the source graph.
    expect(violations).toEqual([{ from: 'domain/invalid', to: 'store/useAuthStore', typeOnly: false }]);
  });

  it('allows erased model imports while rejecting contracts that depend on transport', () => {
    // Given a valid domain model dependency and an invalid contract runtime dependency.
    const edges: readonly ImportEdge[] = [
      { from: 'domain/access', to: 'api/types', typeOnly: true },
      { from: 'api/contracts', to: 'api/client', typeOnly: false },
    ];
    // When both are checked.
    const violations = edges.filter(forbiddenBoundary);
    // Then only the transport dependency reverses the contract boundary.
    expect(violations).toEqual([edges[1]]);
  });
});
