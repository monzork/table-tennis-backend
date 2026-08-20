-- Migration 053 appended a gender symbol to the existing division names
-- ("Primera Division" -> "Primera Division ♂"). That name is used elsewhere
-- as a stable historical key to reassociate a division with its
-- already-played groups/brackets (see bracket.buildGroupEliminationGroups
-- and BuildBracket's division-name-suffix stripping) -- renaming it broke
-- that association for every already-finished event tied to these
-- divisions, corrupting their bracket view. The Gender column (unchanged
-- here) is enough to distinguish them; the display name should stay as it
-- always was.
UPDATE divisions SET name = 'Primera Division' WHERE id = 'div-first';
UPDATE divisions SET name = 'Segunda Division' WHERE id = 'div-second';
UPDATE divisions SET name = 'Tercera Division' WHERE id = 'div-third';
