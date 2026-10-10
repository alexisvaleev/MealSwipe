package shopping

import (
	"testing"
)

func line(id int64, name, cat string, qty float64, unit string) Line {
	return Line{IngredientID: id, Name: name, Category: cat, Quantity: qty, Unit: unit}
}

func part(lines ...Line) Part {
	return Part{RecipeServings: 2, ItemServings: 2, Lines: lines}
}

func find(items []Item, name string) *Item {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}

func TestMergeGramsAndKilos(t *testing.T) {
	items := Generate([]Part{
		part(line(1, "мука", "grains", 500, "g")),
		part(line(1, "мука", "grains", 0.5, "kg")),
	}, nil)
	it := find(items, "мука")
	if it == nil {
		t.Fatal("flour missing")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 merged item, got %d", len(items))
	}
	// 500g + 0.5kg = 1000g -> displayed as 1 kg
	if it.Unit != "kg" || it.Quantity == nil || *it.Quantity != 1.0 {
		t.Fatalf("got %v %s, want 1 kg", it.Quantity, it.Unit)
	}
}

func TestMergeMlAndTbsp(t *testing.T) {
	items := Generate([]Part{
		part(line(2, "молоко", "dairy", 100, "ml")),
		part(line(2, "молоко", "dairy", 2, "tbsp")),
	}, nil)
	it := find(items, "молоко")
	if len(items) != 1 || it == nil {
		t.Fatalf("expected merged milk, got %+v", items)
	}
	// 100ml + 2*15 = 130 ml
	if it.Unit != "ml" || *it.Quantity != 130 {
		t.Fatalf("got %v %s, want 130 ml", *it.Quantity, it.Unit)
	}
}

func TestCountUnitsNotMerged(t *testing.T) {
	items := Generate([]Part{
		part(line(3, "лук", "produce", 2, "pcs")),
		part(line(3, "лук", "produce", 1, "slice")),
	}, nil)
	if len(items) != 2 {
		t.Fatalf("pcs and slice must not merge: %+v", items)
	}
}

func TestCountUnitsMergedWhenIdentical(t *testing.T) {
	items := Generate([]Part{
		part(line(3, "лук", "produce", 2, "pcs")),
		part(line(3, "лук", "produce", 3, "pieces")),
	}, nil)
	if len(items) != 1 || *items[0].Quantity != 5 {
		t.Fatalf("identical count units should merge: %+v", items)
	}
}

func TestPinchUnmerged(t *testing.T) {
	items := Generate([]Part{
		part(line(4, "соль", "spices", 1, "pinch")),
		part(line(4, "соль", "spices", 1, "pinch")),
	}, nil)
	if len(items) != 2 {
		t.Fatalf("pinch lines must stay separate: %+v", items)
	}
}

func TestToTasteLine(t *testing.T) {
	items := Generate([]Part{
		part(line(4, "соль", "spices", 0, "to_taste")),
	}, nil)
	if len(items) != 1 {
		t.Fatalf("want 1 loose line, got %+v", items)
	}
	if items[0].Quantity != nil {
		t.Fatal("to_taste line should carry no merged quantity")
	}
}

func TestPantrySkip(t *testing.T) {
	items := Generate([]Part{
		part(line(1, "мука", "grains", 100, "g"), line(2, "молоко", "dairy", 200, "ml")),
	}, map[int64]bool{1: true})
	if len(items) != 1 || items[0].Name != "молоко" {
		t.Fatalf("pantry ingredient not skipped: %+v", items)
	}
}

func TestServingsScaling(t *testing.T) {
	p := Part{RecipeServings: 2, ItemServings: 4, Lines: []Line{
		line(1, "мука", "grains", 100, "g"),
	}}
	items := Generate([]Part{p}, nil)
	if len(items) != 1 || *items[0].Quantity != 200 {
		t.Fatalf("expected 200g after doubling servings: %+v", items)
	}
}

func TestKgDisplay(t *testing.T) {
	items := Generate([]Part{
		part(line(1, "сахар", "other", 1500, "g")),
	}, nil)
	if items[0].Unit != "kg" || *items[0].Quantity != 1.5 {
		t.Fatalf("expected 1.5 kg, got %v %s", *items[0].Quantity, items[0].Unit)
	}
}

func TestLiterDisplay(t *testing.T) {
	items := Generate([]Part{
		part(line(2, "молоко", "dairy", 1200, "ml")),
	}, nil)
	if items[0].Unit != "l" || *items[0].Quantity != 1.2 {
		t.Fatalf("expected 1.2 l, got %v %s", *items[0].Quantity, items[0].Unit)
	}
}

func TestZeroQtyUnmerged(t *testing.T) {
	items := Generate([]Part{
		part(line(4, "соль", "spices", 0, "g")),
		part(line(4, "соль", "spices", 0, "g")),
	}, nil)
	if len(items) != 2 {
		t.Fatalf("qty=0 lines must be unmerged: %+v", items)
	}
}

func TestCategoryGrouping(t *testing.T) {
	items := Generate([]Part{
		part(
			line(9, "хлеб", "bakery", 1, "pcs"),
			line(1, "морковь", "produce", 3, "pcs"),
			line(2, "молоко", "dairy", 200, "ml"),
		),
	}, nil)
	// produce(0) < dairy(1) < bakery(4)
	if items[0].Category != "produce" || items[1].Category != "dairy" || items[2].Category != "bakery" {
		t.Fatalf("not grouped by category order: %+v", items)
	}
	for i, it := range items {
		if it.Position != i {
			t.Fatalf("bad position on %+v", it)
		}
	}
}
