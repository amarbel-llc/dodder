wit_bindgen::generate!({
    world: "query-filter",
    path: "../../../wit",
});

struct GenreFilter;

impl Guest for GenreFilter {
    fn contains_sku(object: Sku) -> bool {
        object.genre == "zettel"
    }
}

export!(GenreFilter);
