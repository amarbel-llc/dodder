#include "tree_sitter/parser.h"

#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic ignored "-Wmissing-field-initializers"
#endif

#define LANGUAGE_VERSION 14
#define STATE_COUNT 92
#define LARGE_STATE_COUNT 2
#define SYMBOL_COUNT 63
#define ALIAS_COUNT 0
#define TOKEN_COUNT 29
#define EXTERNAL_TOKEN_COUNT 0
#define FIELD_COUNT 14
#define MAX_ALIAS_SEQUENCE_LENGTH 6
#define PRODUCTION_ID_COUNT 15

enum ts_symbol_identifiers {
  anon_sym_LF = 1,
  anon_sym_SPACE = 2,
  sym_heading_marker = 3,
  sym__heading_comma = 4,
  sym_heading_tag = 5,
  anon_sym_DASH = 6,
  anon_sym_PERCENT = 7,
  aux_sym_description_token1 = 8,
  anon_sym_DASH_DASH_DASH = 9,
  anon_sym_POUND = 10,
  anon_sym_LT = 11,
  anon_sym_BANG = 12,
  anon_sym_AT = 13,
  sym_type_name = 14,
  sym_file_path = 15,
  anon_sym_LBRACK = 16,
  anon_sym_RBRACK = 17,
  sym__box_space = 18,
  anon_sym_EQ = 19,
  sym_box_bare_value = 20,
  anon_sym_SLASH = 21,
  sym_box_ident = 22,
  anon_sym_DQUOTE = 23,
  aux_sym_box_quoted_token1 = 24,
  sym_box_escape = 25,
  sym_markl_purpose = 26,
  sym_markl_format = 27,
  sym_markl_data = 28,
  sym_source_file = 29,
  sym__body_line = 30,
  sym_blank_line = 31,
  sym_heading = 32,
  sym_object_line = 33,
  sym_description = 34,
  sym_metadata = 35,
  sym__metadata_line = 36,
  sym_description_line = 37,
  sym_description_text = 38,
  sym_tag_line = 39,
  sym_tag_name = 40,
  sym_ref_line = 41,
  sym_reference_id = 42,
  sym_type_line = 43,
  sym_blob_line = 44,
  sym_comment_line = 45,
  sym_comment_text = 46,
  sym_box = 47,
  sym__box_item = 48,
  sym_box_blob = 49,
  sym_box_type = 50,
  sym_box_computed_tag = 51,
  sym_box_field = 52,
  sym__box_value = 53,
  sym_box_object_id = 54,
  sym_box_tag = 55,
  sym_box_quoted = 56,
  sym_markl_id = 57,
  aux_sym_source_file_repeat1 = 58,
  aux_sym_heading_repeat1 = 59,
  aux_sym_metadata_repeat1 = 60,
  aux_sym_box_repeat1 = 61,
  aux_sym_box_quoted_repeat1 = 62,
};

static const char * const ts_symbol_names[] = {
  [ts_builtin_sym_end] = "end",
  [anon_sym_LF] = "\n",
  [anon_sym_SPACE] = " ",
  [sym_heading_marker] = "heading_marker",
  [sym__heading_comma] = "_heading_comma",
  [sym_heading_tag] = "heading_tag",
  [anon_sym_DASH] = "-",
  [anon_sym_PERCENT] = "%",
  [aux_sym_description_token1] = "description_token1",
  [anon_sym_DASH_DASH_DASH] = "---",
  [anon_sym_POUND] = "#",
  [anon_sym_LT] = "<",
  [anon_sym_BANG] = "!",
  [anon_sym_AT] = "@",
  [sym_type_name] = "type_name",
  [sym_file_path] = "file_path",
  [anon_sym_LBRACK] = "[",
  [anon_sym_RBRACK] = "]",
  [sym__box_space] = "_box_space",
  [anon_sym_EQ] = "=",
  [sym_box_bare_value] = "box_bare_value",
  [anon_sym_SLASH] = "/",
  [sym_box_ident] = "box_ident",
  [anon_sym_DQUOTE] = "\"",
  [aux_sym_box_quoted_token1] = "box_quoted_token1",
  [sym_box_escape] = "box_escape",
  [sym_markl_purpose] = "markl_purpose",
  [sym_markl_format] = "markl_format",
  [sym_markl_data] = "markl_data",
  [sym_source_file] = "source_file",
  [sym__body_line] = "_body_line",
  [sym_blank_line] = "blank_line",
  [sym_heading] = "heading",
  [sym_object_line] = "object_line",
  [sym_description] = "description",
  [sym_metadata] = "metadata",
  [sym__metadata_line] = "_metadata_line",
  [sym_description_line] = "description_line",
  [sym_description_text] = "description_text",
  [sym_tag_line] = "tag_line",
  [sym_tag_name] = "tag_name",
  [sym_ref_line] = "ref_line",
  [sym_reference_id] = "reference_id",
  [sym_type_line] = "type_line",
  [sym_blob_line] = "blob_line",
  [sym_comment_line] = "comment_line",
  [sym_comment_text] = "comment_text",
  [sym_box] = "box",
  [sym__box_item] = "_box_item",
  [sym_box_blob] = "box_blob",
  [sym_box_type] = "box_type",
  [sym_box_computed_tag] = "box_computed_tag",
  [sym_box_field] = "box_field",
  [sym__box_value] = "_box_value",
  [sym_box_object_id] = "box_object_id",
  [sym_box_tag] = "box_tag",
  [sym_box_quoted] = "box_quoted",
  [sym_markl_id] = "markl_id",
  [aux_sym_source_file_repeat1] = "source_file_repeat1",
  [aux_sym_heading_repeat1] = "heading_repeat1",
  [aux_sym_metadata_repeat1] = "metadata_repeat1",
  [aux_sym_box_repeat1] = "box_repeat1",
  [aux_sym_box_quoted_repeat1] = "box_quoted_repeat1",
};

static const TSSymbol ts_symbol_map[] = {
  [ts_builtin_sym_end] = ts_builtin_sym_end,
  [anon_sym_LF] = anon_sym_LF,
  [anon_sym_SPACE] = anon_sym_SPACE,
  [sym_heading_marker] = sym_heading_marker,
  [sym__heading_comma] = sym__heading_comma,
  [sym_heading_tag] = sym_heading_tag,
  [anon_sym_DASH] = anon_sym_DASH,
  [anon_sym_PERCENT] = anon_sym_PERCENT,
  [aux_sym_description_token1] = aux_sym_description_token1,
  [anon_sym_DASH_DASH_DASH] = anon_sym_DASH_DASH_DASH,
  [anon_sym_POUND] = anon_sym_POUND,
  [anon_sym_LT] = anon_sym_LT,
  [anon_sym_BANG] = anon_sym_BANG,
  [anon_sym_AT] = anon_sym_AT,
  [sym_type_name] = sym_type_name,
  [sym_file_path] = sym_file_path,
  [anon_sym_LBRACK] = anon_sym_LBRACK,
  [anon_sym_RBRACK] = anon_sym_RBRACK,
  [sym__box_space] = sym__box_space,
  [anon_sym_EQ] = anon_sym_EQ,
  [sym_box_bare_value] = sym_box_bare_value,
  [anon_sym_SLASH] = anon_sym_SLASH,
  [sym_box_ident] = sym_box_ident,
  [anon_sym_DQUOTE] = anon_sym_DQUOTE,
  [aux_sym_box_quoted_token1] = aux_sym_box_quoted_token1,
  [sym_box_escape] = sym_box_escape,
  [sym_markl_purpose] = sym_markl_purpose,
  [sym_markl_format] = sym_markl_format,
  [sym_markl_data] = sym_markl_data,
  [sym_source_file] = sym_source_file,
  [sym__body_line] = sym__body_line,
  [sym_blank_line] = sym_blank_line,
  [sym_heading] = sym_heading,
  [sym_object_line] = sym_object_line,
  [sym_description] = sym_description,
  [sym_metadata] = sym_metadata,
  [sym__metadata_line] = sym__metadata_line,
  [sym_description_line] = sym_description_line,
  [sym_description_text] = sym_description_text,
  [sym_tag_line] = sym_tag_line,
  [sym_tag_name] = sym_tag_name,
  [sym_ref_line] = sym_ref_line,
  [sym_reference_id] = sym_reference_id,
  [sym_type_line] = sym_type_line,
  [sym_blob_line] = sym_blob_line,
  [sym_comment_line] = sym_comment_line,
  [sym_comment_text] = sym_comment_text,
  [sym_box] = sym_box,
  [sym__box_item] = sym__box_item,
  [sym_box_blob] = sym_box_blob,
  [sym_box_type] = sym_box_type,
  [sym_box_computed_tag] = sym_box_computed_tag,
  [sym_box_field] = sym_box_field,
  [sym__box_value] = sym__box_value,
  [sym_box_object_id] = sym_box_object_id,
  [sym_box_tag] = sym_box_tag,
  [sym_box_quoted] = sym_box_quoted,
  [sym_markl_id] = sym_markl_id,
  [aux_sym_source_file_repeat1] = aux_sym_source_file_repeat1,
  [aux_sym_heading_repeat1] = aux_sym_heading_repeat1,
  [aux_sym_metadata_repeat1] = aux_sym_metadata_repeat1,
  [aux_sym_box_repeat1] = aux_sym_box_repeat1,
  [aux_sym_box_quoted_repeat1] = aux_sym_box_quoted_repeat1,
};

static const TSSymbolMetadata ts_symbol_metadata[] = {
  [ts_builtin_sym_end] = {
    .visible = false,
    .named = true,
  },
  [anon_sym_LF] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SPACE] = {
    .visible = true,
    .named = false,
  },
  [sym_heading_marker] = {
    .visible = true,
    .named = true,
  },
  [sym__heading_comma] = {
    .visible = false,
    .named = true,
  },
  [sym_heading_tag] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_DASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_PERCENT] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_description_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_DASH_DASH_DASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUND] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LT] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_BANG] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_AT] = {
    .visible = true,
    .named = false,
  },
  [sym_type_name] = {
    .visible = true,
    .named = true,
  },
  [sym_file_path] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_LBRACK] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_RBRACK] = {
    .visible = true,
    .named = false,
  },
  [sym__box_space] = {
    .visible = false,
    .named = true,
  },
  [anon_sym_EQ] = {
    .visible = true,
    .named = false,
  },
  [sym_box_bare_value] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_SLASH] = {
    .visible = true,
    .named = false,
  },
  [sym_box_ident] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_DQUOTE] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_box_quoted_token1] = {
    .visible = false,
    .named = false,
  },
  [sym_box_escape] = {
    .visible = true,
    .named = true,
  },
  [sym_markl_purpose] = {
    .visible = true,
    .named = true,
  },
  [sym_markl_format] = {
    .visible = true,
    .named = true,
  },
  [sym_markl_data] = {
    .visible = true,
    .named = true,
  },
  [sym_source_file] = {
    .visible = true,
    .named = true,
  },
  [sym__body_line] = {
    .visible = false,
    .named = true,
  },
  [sym_blank_line] = {
    .visible = true,
    .named = true,
  },
  [sym_heading] = {
    .visible = true,
    .named = true,
  },
  [sym_object_line] = {
    .visible = true,
    .named = true,
  },
  [sym_description] = {
    .visible = true,
    .named = true,
  },
  [sym_metadata] = {
    .visible = true,
    .named = true,
  },
  [sym__metadata_line] = {
    .visible = false,
    .named = true,
  },
  [sym_description_line] = {
    .visible = true,
    .named = true,
  },
  [sym_description_text] = {
    .visible = true,
    .named = true,
  },
  [sym_tag_line] = {
    .visible = true,
    .named = true,
  },
  [sym_tag_name] = {
    .visible = true,
    .named = true,
  },
  [sym_ref_line] = {
    .visible = true,
    .named = true,
  },
  [sym_reference_id] = {
    .visible = true,
    .named = true,
  },
  [sym_type_line] = {
    .visible = true,
    .named = true,
  },
  [sym_blob_line] = {
    .visible = true,
    .named = true,
  },
  [sym_comment_line] = {
    .visible = true,
    .named = true,
  },
  [sym_comment_text] = {
    .visible = true,
    .named = true,
  },
  [sym_box] = {
    .visible = true,
    .named = true,
  },
  [sym__box_item] = {
    .visible = false,
    .named = true,
  },
  [sym_box_blob] = {
    .visible = true,
    .named = true,
  },
  [sym_box_type] = {
    .visible = true,
    .named = true,
  },
  [sym_box_computed_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_box_field] = {
    .visible = true,
    .named = true,
  },
  [sym__box_value] = {
    .visible = false,
    .named = true,
  },
  [sym_box_object_id] = {
    .visible = true,
    .named = true,
  },
  [sym_box_tag] = {
    .visible = true,
    .named = true,
  },
  [sym_box_quoted] = {
    .visible = true,
    .named = true,
  },
  [sym_markl_id] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_source_file_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_heading_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_metadata_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_box_repeat1] = {
    .visible = false,
    .named = false,
  },
  [aux_sym_box_quoted_repeat1] = {
    .visible = false,
    .named = false,
  },
};

enum ts_field_identifiers {
  field_data = 1,
  field_description = 2,
  field_format = 3,
  field_key = 4,
  field_left = 5,
  field_lock = 6,
  field_marker = 7,
  field_name = 8,
  field_prefix = 9,
  field_ref = 10,
  field_right = 11,
  field_text = 12,
  field_type = 13,
  field_value = 14,
};

static const char * const ts_field_names[] = {
  [0] = NULL,
  [field_data] = "data",
  [field_description] = "description",
  [field_format] = "format",
  [field_key] = "key",
  [field_left] = "left",
  [field_lock] = "lock",
  [field_marker] = "marker",
  [field_name] = "name",
  [field_prefix] = "prefix",
  [field_ref] = "ref",
  [field_right] = "right",
  [field_text] = "text",
  [field_type] = "type",
  [field_value] = "value",
};

static const TSFieldMapSlice ts_field_map_slices[PRODUCTION_ID_COUNT] = {
  [1] = {.index = 0, .length = 1},
  [2] = {.index = 1, .length = 1},
  [3] = {.index = 2, .length = 1},
  [4] = {.index = 3, .length = 1},
  [5] = {.index = 4, .length = 2},
  [6] = {.index = 6, .length = 2},
  [7] = {.index = 8, .length = 2},
  [8] = {.index = 10, .length = 1},
  [9] = {.index = 11, .length = 1},
  [10] = {.index = 12, .length = 1},
  [11] = {.index = 13, .length = 1},
  [12] = {.index = 14, .length = 2},
  [13] = {.index = 16, .length = 2},
  [14] = {.index = 18, .length = 2},
};

static const TSFieldMapEntry ts_field_map_entries[] = {
  [0] =
    {field_marker, 0},
  [1] =
    {field_prefix, 0},
  [2] =
    {field_name, 1},
  [3] =
    {field_text, 1},
  [4] =
    {field_key, 0},
    {field_value, 2},
  [6] =
    {field_left, 0},
    {field_right, 2},
  [8] =
    {field_description, 4},
    {field_prefix, 0},
  [10] =
    {field_name, 2},
  [11] =
    {field_text, 2},
  [12] =
    {field_ref, 2},
  [13] =
    {field_type, 2},
  [14] =
    {field_data, 2},
    {field_format, 0},
  [16] =
    {field_data, 3},
    {field_format, 1},
  [18] =
    {field_lock, 4},
    {field_type, 2},
};

static const TSSymbol ts_alias_sequences[PRODUCTION_ID_COUNT][MAX_ALIAS_SEQUENCE_LENGTH] = {
  [0] = {0},
};

static const uint16_t ts_non_terminal_alias_map[] = {
  0,
};

static const TSStateId ts_primary_state_ids[STATE_COUNT] = {
  [0] = 0,
  [1] = 1,
  [2] = 2,
  [3] = 3,
  [4] = 4,
  [5] = 5,
  [6] = 6,
  [7] = 7,
  [8] = 8,
  [9] = 9,
  [10] = 10,
  [11] = 11,
  [12] = 12,
  [13] = 13,
  [14] = 14,
  [15] = 15,
  [16] = 16,
  [17] = 17,
  [18] = 18,
  [19] = 19,
  [20] = 20,
  [21] = 21,
  [22] = 22,
  [23] = 23,
  [24] = 24,
  [25] = 25,
  [26] = 26,
  [27] = 27,
  [28] = 28,
  [29] = 29,
  [30] = 30,
  [31] = 31,
  [32] = 32,
  [33] = 33,
  [34] = 34,
  [35] = 35,
  [36] = 36,
  [37] = 37,
  [38] = 38,
  [39] = 39,
  [40] = 40,
  [41] = 41,
  [42] = 42,
  [43] = 43,
  [44] = 44,
  [45] = 45,
  [46] = 46,
  [47] = 47,
  [48] = 48,
  [49] = 49,
  [50] = 50,
  [51] = 51,
  [52] = 52,
  [53] = 53,
  [54] = 54,
  [55] = 55,
  [56] = 56,
  [57] = 57,
  [58] = 58,
  [59] = 59,
  [60] = 60,
  [61] = 61,
  [62] = 62,
  [63] = 63,
  [64] = 64,
  [65] = 65,
  [66] = 66,
  [67] = 67,
  [68] = 68,
  [69] = 69,
  [70] = 70,
  [71] = 71,
  [72] = 72,
  [73] = 73,
  [74] = 74,
  [75] = 75,
  [76] = 76,
  [77] = 77,
  [78] = 78,
  [79] = 79,
  [80] = 80,
  [81] = 81,
  [82] = 82,
  [83] = 83,
  [84] = 84,
  [85] = 85,
  [86] = 86,
  [87] = 87,
  [88] = 88,
  [89] = 89,
  [90] = 90,
  [91] = 91,
};

static bool ts_lex(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      if (eof) ADVANCE(17);
      ADVANCE_MAP(
        '\n', 18,
        ' ', 19,
        '!', 30,
        '"', 43,
        '#', 28,
        '%', 24,
        ',', 21,
        '-', 23,
        '/', 39,
        '<', 29,
        '=', 37,
        '@', 31,
        '[', 34,
        '\\', 15,
        ']', 35,
        '_', 49,
      );
      if (('A' <= lookahead && lookahead <= 'Z')) ADVANCE(42);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(40);
      END_STATE();
    case 1:
      if (lookahead == '\n') ADVANCE(18);
      if (lookahead != 0) ADVANCE(25);
      END_STATE();
    case 2:
      if (lookahead == '!') ADVANCE(30);
      if (lookahead == '#') ADVANCE(27);
      if (lookahead == '%') ADVANCE(24);
      if (lookahead == '-') ADVANCE(23);
      if (lookahead == '<') ADVANCE(29);
      if (lookahead == '@') ADVANCE(31);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(47);
      END_STATE();
    case 3:
      if (lookahead == '"') ADVANCE(43);
      if (lookahead == '\\') ADVANCE(15);
      if (lookahead != 0) ADVANCE(44);
      END_STATE();
    case 4:
      if (lookahead == '"') ADVANCE(43);
      if (lookahead != 0 &&
          lookahead != '\t' &&
          lookahead != ' ' &&
          lookahead != ']') ADVANCE(38);
      END_STATE();
    case 5:
      if (lookahead == '-') ADVANCE(26);
      END_STATE();
    case 6:
      if (lookahead == '.') ADVANCE(8);
      if (lookahead == '@') ADVANCE(46);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(6);
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(9);
      END_STATE();
    case 7:
      if (lookahead == '.') ADVANCE(8);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(48);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(9);
      END_STATE();
    case 8:
      if (lookahead == '.') ADVANCE(8);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(33);
      END_STATE();
    case 9:
      if (lookahead == '.') ADVANCE(8);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(9);
      END_STATE();
    case 10:
      if (lookahead == '@') ADVANCE(46);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(10);
      END_STATE();
    case 11:
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(50);
      END_STATE();
    case 12:
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(49);
      END_STATE();
    case 13:
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != ',') ADVANCE(22);
      END_STATE();
    case 14:
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(32);
      END_STATE();
    case 15:
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(45);
      END_STATE();
    case 16:
      if (eof) ADVANCE(17);
      ADVANCE_MAP(
        '\n', 18,
        '!', 30,
        '"', 43,
        '#', 20,
        '%', 24,
        '-', 23,
        '/', 39,
        '=', 37,
        '@', 31,
        ']', 35,
        '\t', 36,
        ' ', 36,
      );
      if (('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(42);
      END_STATE();
    case 17:
      ACCEPT_TOKEN(ts_builtin_sym_end);
      END_STATE();
    case 18:
      ACCEPT_TOKEN(anon_sym_LF);
      END_STATE();
    case 19:
      ACCEPT_TOKEN(anon_sym_SPACE);
      END_STATE();
    case 20:
      ACCEPT_TOKEN(sym_heading_marker);
      if (lookahead == '#') ADVANCE(20);
      END_STATE();
    case 21:
      ACCEPT_TOKEN(sym__heading_comma);
      if (lookahead == '\t' ||
          lookahead == ' ') ADVANCE(21);
      END_STATE();
    case 22:
      ACCEPT_TOKEN(sym_heading_tag);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != ',') ADVANCE(22);
      END_STATE();
    case 23:
      ACCEPT_TOKEN(anon_sym_DASH);
      if (lookahead == '-') ADVANCE(5);
      END_STATE();
    case 24:
      ACCEPT_TOKEN(anon_sym_PERCENT);
      END_STATE();
    case 25:
      ACCEPT_TOKEN(aux_sym_description_token1);
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(25);
      END_STATE();
    case 26:
      ACCEPT_TOKEN(anon_sym_DASH_DASH_DASH);
      END_STATE();
    case 27:
      ACCEPT_TOKEN(anon_sym_POUND);
      END_STATE();
    case 28:
      ACCEPT_TOKEN(anon_sym_POUND);
      if (lookahead == '#') ADVANCE(20);
      END_STATE();
    case 29:
      ACCEPT_TOKEN(anon_sym_LT);
      END_STATE();
    case 30:
      ACCEPT_TOKEN(anon_sym_BANG);
      END_STATE();
    case 31:
      ACCEPT_TOKEN(anon_sym_AT);
      END_STATE();
    case 32:
      ACCEPT_TOKEN(sym_type_name);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(32);
      END_STATE();
    case 33:
      ACCEPT_TOKEN(sym_file_path);
      if (lookahead == '.') ADVANCE(8);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(33);
      END_STATE();
    case 34:
      ACCEPT_TOKEN(anon_sym_LBRACK);
      END_STATE();
    case 35:
      ACCEPT_TOKEN(anon_sym_RBRACK);
      END_STATE();
    case 36:
      ACCEPT_TOKEN(sym__box_space);
      if (lookahead == '\t' ||
          lookahead == ' ') ADVANCE(36);
      END_STATE();
    case 37:
      ACCEPT_TOKEN(anon_sym_EQ);
      END_STATE();
    case 38:
      ACCEPT_TOKEN(sym_box_bare_value);
      if (lookahead != 0 &&
          lookahead != '\t' &&
          lookahead != ' ' &&
          lookahead != '"' &&
          lookahead != ']') ADVANCE(38);
      END_STATE();
    case 39:
      ACCEPT_TOKEN(anon_sym_SLASH);
      END_STATE();
    case 40:
      ACCEPT_TOKEN(sym_box_ident);
      if (lookahead == '_') ADVANCE(41);
      if (lookahead == '-' ||
          ('A' <= lookahead && lookahead <= 'Z')) ADVANCE(42);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(40);
      END_STATE();
    case 41:
      ACCEPT_TOKEN(sym_box_ident);
      if (lookahead == '-' ||
          ('A' <= lookahead && lookahead <= 'Z')) ADVANCE(42);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(41);
      END_STATE();
    case 42:
      ACCEPT_TOKEN(sym_box_ident);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          ('A' <= lookahead && lookahead <= 'Z') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(42);
      END_STATE();
    case 43:
      ACCEPT_TOKEN(anon_sym_DQUOTE);
      END_STATE();
    case 44:
      ACCEPT_TOKEN(aux_sym_box_quoted_token1);
      if (lookahead != 0 &&
          lookahead != '"' &&
          lookahead != '\\') ADVANCE(44);
      END_STATE();
    case 45:
      ACCEPT_TOKEN(sym_box_escape);
      END_STATE();
    case 46:
      ACCEPT_TOKEN(sym_markl_purpose);
      END_STATE();
    case 47:
      ACCEPT_TOKEN(sym_markl_format);
      if (lookahead == '-') ADVANCE(10);
      if (lookahead == '@') ADVANCE(46);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(47);
      END_STATE();
    case 48:
      ACCEPT_TOKEN(sym_markl_format);
      if (lookahead == '-') ADVANCE(6);
      if (lookahead == '.') ADVANCE(8);
      if (lookahead == '@') ADVANCE(46);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(48);
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(9);
      END_STATE();
    case 49:
      ACCEPT_TOKEN(sym_markl_format);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(49);
      END_STATE();
    case 50:
      ACCEPT_TOKEN(sym_markl_data);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(50);
      END_STATE();
    default:
      return false;
  }
}

static const TSLexMode ts_lex_modes[STATE_COUNT] = {
  [0] = {.lex_state = 0},
  [1] = {.lex_state = 16},
  [2] = {.lex_state = 16},
  [3] = {.lex_state = 16},
  [4] = {.lex_state = 16},
  [5] = {.lex_state = 2},
  [6] = {.lex_state = 2},
  [7] = {.lex_state = 2},
  [8] = {.lex_state = 16},
  [9] = {.lex_state = 16},
  [10] = {.lex_state = 16},
  [11] = {.lex_state = 16},
  [12] = {.lex_state = 16},
  [13] = {.lex_state = 16},
  [14] = {.lex_state = 16},
  [15] = {.lex_state = 16},
  [16] = {.lex_state = 16},
  [17] = {.lex_state = 16},
  [18] = {.lex_state = 16},
  [19] = {.lex_state = 16},
  [20] = {.lex_state = 16},
  [21] = {.lex_state = 16},
  [22] = {.lex_state = 16},
  [23] = {.lex_state = 2},
  [24] = {.lex_state = 2},
  [25] = {.lex_state = 2},
  [26] = {.lex_state = 2},
  [27] = {.lex_state = 2},
  [28] = {.lex_state = 2},
  [29] = {.lex_state = 2},
  [30] = {.lex_state = 2},
  [31] = {.lex_state = 16},
  [32] = {.lex_state = 16},
  [33] = {.lex_state = 16},
  [34] = {.lex_state = 16},
  [35] = {.lex_state = 16},
  [36] = {.lex_state = 16},
  [37] = {.lex_state = 16},
  [38] = {.lex_state = 4},
  [39] = {.lex_state = 7},
  [40] = {.lex_state = 3},
  [41] = {.lex_state = 3},
  [42] = {.lex_state = 3},
  [43] = {.lex_state = 0},
  [44] = {.lex_state = 0},
  [45] = {.lex_state = 2},
  [46] = {.lex_state = 0},
  [47] = {.lex_state = 2},
  [48] = {.lex_state = 1},
  [49] = {.lex_state = 1},
  [50] = {.lex_state = 0},
  [51] = {.lex_state = 1},
  [52] = {.lex_state = 0},
  [53] = {.lex_state = 0},
  [54] = {.lex_state = 0},
  [55] = {.lex_state = 0},
  [56] = {.lex_state = 0},
  [57] = {.lex_state = 1},
  [58] = {.lex_state = 1},
  [59] = {.lex_state = 11},
  [60] = {.lex_state = 13},
  [61] = {.lex_state = 0},
  [62] = {.lex_state = 0},
  [63] = {.lex_state = 0},
  [64] = {.lex_state = 0},
  [65] = {.lex_state = 0},
  [66] = {.lex_state = 0},
  [67] = {.lex_state = 16},
  [68] = {.lex_state = 0},
  [69] = {.lex_state = 0},
  [70] = {.lex_state = 0},
  [71] = {.lex_state = 0},
  [72] = {.lex_state = 0},
  [73] = {.lex_state = 0},
  [74] = {.lex_state = 0},
  [75] = {.lex_state = 11},
  [76] = {.lex_state = 0},
  [77] = {.lex_state = 0},
  [78] = {.lex_state = 0},
  [79] = {.lex_state = 0},
  [80] = {.lex_state = 12},
  [81] = {.lex_state = 0},
  [82] = {.lex_state = 13},
  [83] = {.lex_state = 16},
  [84] = {.lex_state = 0},
  [85] = {.lex_state = 14},
  [86] = {.lex_state = 16},
  [87] = {.lex_state = 0},
  [88] = {.lex_state = 0},
  [89] = {.lex_state = 0},
  [90] = {.lex_state = 0},
  [91] = {.lex_state = 16},
};

static const uint16_t ts_parse_table[LARGE_STATE_COUNT][SYMBOL_COUNT] = {
  [0] = {
    [ts_builtin_sym_end] = ACTIONS(1),
    [anon_sym_LF] = ACTIONS(1),
    [anon_sym_SPACE] = ACTIONS(1),
    [sym_heading_marker] = ACTIONS(1),
    [sym__heading_comma] = ACTIONS(1),
    [anon_sym_DASH] = ACTIONS(1),
    [anon_sym_PERCENT] = ACTIONS(1),
    [anon_sym_DASH_DASH_DASH] = ACTIONS(1),
    [anon_sym_POUND] = ACTIONS(1),
    [anon_sym_LT] = ACTIONS(1),
    [anon_sym_BANG] = ACTIONS(1),
    [anon_sym_AT] = ACTIONS(1),
    [anon_sym_LBRACK] = ACTIONS(1),
    [anon_sym_RBRACK] = ACTIONS(1),
    [anon_sym_EQ] = ACTIONS(1),
    [anon_sym_SLASH] = ACTIONS(1),
    [sym_box_ident] = ACTIONS(1),
    [anon_sym_DQUOTE] = ACTIONS(1),
    [sym_box_escape] = ACTIONS(1),
    [sym_markl_format] = ACTIONS(1),
    [sym_markl_data] = ACTIONS(1),
  },
  [1] = {
    [sym_source_file] = STATE(79),
    [sym__body_line] = STATE(8),
    [sym_blank_line] = STATE(8),
    [sym_heading] = STATE(8),
    [sym_object_line] = STATE(8),
    [sym_metadata] = STATE(11),
    [aux_sym_source_file_repeat1] = STATE(8),
    [ts_builtin_sym_end] = ACTIONS(3),
    [anon_sym_LF] = ACTIONS(5),
    [sym_heading_marker] = ACTIONS(7),
    [anon_sym_DASH] = ACTIONS(9),
    [anon_sym_PERCENT] = ACTIONS(11),
    [anon_sym_DASH_DASH_DASH] = ACTIONS(13),
  },
};

static const uint16_t ts_small_parse_table[] = {
  [0] = 9,
    ACTIONS(15), 1,
      anon_sym_PERCENT,
    ACTIONS(17), 1,
      anon_sym_BANG,
    ACTIONS(19), 1,
      anon_sym_AT,
    ACTIONS(21), 1,
      anon_sym_RBRACK,
    ACTIONS(23), 1,
      sym__box_space,
    ACTIONS(25), 1,
      anon_sym_SLASH,
    ACTIONS(27), 1,
      sym_box_ident,
    ACTIONS(29), 1,
      anon_sym_DQUOTE,
    STATE(3), 9,
      sym__box_item,
      sym_box_blob,
      sym_box_type,
      sym_box_computed_tag,
      sym_box_field,
      sym_box_object_id,
      sym_box_tag,
      sym_box_quoted,
      aux_sym_box_repeat1,
  [36] = 9,
    ACTIONS(15), 1,
      anon_sym_PERCENT,
    ACTIONS(17), 1,
      anon_sym_BANG,
    ACTIONS(19), 1,
      anon_sym_AT,
    ACTIONS(25), 1,
      anon_sym_SLASH,
    ACTIONS(27), 1,
      sym_box_ident,
    ACTIONS(29), 1,
      anon_sym_DQUOTE,
    ACTIONS(31), 1,
      anon_sym_RBRACK,
    ACTIONS(33), 1,
      sym__box_space,
    STATE(4), 9,
      sym__box_item,
      sym_box_blob,
      sym_box_type,
      sym_box_computed_tag,
      sym_box_field,
      sym_box_object_id,
      sym_box_tag,
      sym_box_quoted,
      aux_sym_box_repeat1,
  [72] = 9,
    ACTIONS(35), 1,
      anon_sym_PERCENT,
    ACTIONS(38), 1,
      anon_sym_BANG,
    ACTIONS(41), 1,
      anon_sym_AT,
    ACTIONS(44), 1,
      anon_sym_RBRACK,
    ACTIONS(46), 1,
      sym__box_space,
    ACTIONS(49), 1,
      anon_sym_SLASH,
    ACTIONS(52), 1,
      sym_box_ident,
    ACTIONS(55), 1,
      anon_sym_DQUOTE,
    STATE(4), 9,
      sym__box_item,
      sym_box_blob,
      sym_box_type,
      sym_box_computed_tag,
      sym_box_field,
      sym_box_object_id,
      sym_box_tag,
      sym_box_quoted,
      aux_sym_box_repeat1,
  [108] = 8,
    ACTIONS(58), 1,
      anon_sym_DASH,
    ACTIONS(60), 1,
      anon_sym_PERCENT,
    ACTIONS(62), 1,
      anon_sym_DASH_DASH_DASH,
    ACTIONS(64), 1,
      anon_sym_POUND,
    ACTIONS(66), 1,
      anon_sym_LT,
    ACTIONS(68), 1,
      anon_sym_BANG,
    ACTIONS(70), 1,
      anon_sym_AT,
    STATE(7), 8,
      sym__metadata_line,
      sym_description_line,
      sym_tag_line,
      sym_ref_line,
      sym_type_line,
      sym_blob_line,
      sym_comment_line,
      aux_sym_metadata_repeat1,
  [140] = 8,
    ACTIONS(58), 1,
      anon_sym_DASH,
    ACTIONS(60), 1,
      anon_sym_PERCENT,
    ACTIONS(64), 1,
      anon_sym_POUND,
    ACTIONS(66), 1,
      anon_sym_LT,
    ACTIONS(68), 1,
      anon_sym_BANG,
    ACTIONS(70), 1,
      anon_sym_AT,
    ACTIONS(72), 1,
      anon_sym_DASH_DASH_DASH,
    STATE(5), 8,
      sym__metadata_line,
      sym_description_line,
      sym_tag_line,
      sym_ref_line,
      sym_type_line,
      sym_blob_line,
      sym_comment_line,
      aux_sym_metadata_repeat1,
  [172] = 8,
    ACTIONS(74), 1,
      anon_sym_DASH,
    ACTIONS(77), 1,
      anon_sym_PERCENT,
    ACTIONS(80), 1,
      anon_sym_DASH_DASH_DASH,
    ACTIONS(82), 1,
      anon_sym_POUND,
    ACTIONS(85), 1,
      anon_sym_LT,
    ACTIONS(88), 1,
      anon_sym_BANG,
    ACTIONS(91), 1,
      anon_sym_AT,
    STATE(7), 8,
      sym__metadata_line,
      sym_description_line,
      sym_tag_line,
      sym_ref_line,
      sym_type_line,
      sym_blob_line,
      sym_comment_line,
      aux_sym_metadata_repeat1,
  [204] = 5,
    ACTIONS(5), 1,
      anon_sym_LF,
    ACTIONS(7), 1,
      sym_heading_marker,
    ACTIONS(94), 1,
      ts_builtin_sym_end,
    ACTIONS(11), 2,
      anon_sym_DASH,
      anon_sym_PERCENT,
    STATE(10), 5,
      sym__body_line,
      sym_blank_line,
      sym_heading,
      sym_object_line,
      aux_sym_source_file_repeat1,
  [225] = 5,
    ACTIONS(5), 1,
      anon_sym_LF,
    ACTIONS(7), 1,
      sym_heading_marker,
    ACTIONS(96), 1,
      ts_builtin_sym_end,
    ACTIONS(11), 2,
      anon_sym_DASH,
      anon_sym_PERCENT,
    STATE(10), 5,
      sym__body_line,
      sym_blank_line,
      sym_heading,
      sym_object_line,
      aux_sym_source_file_repeat1,
  [246] = 5,
    ACTIONS(98), 1,
      ts_builtin_sym_end,
    ACTIONS(100), 1,
      anon_sym_LF,
    ACTIONS(103), 1,
      sym_heading_marker,
    ACTIONS(106), 2,
      anon_sym_DASH,
      anon_sym_PERCENT,
    STATE(10), 5,
      sym__body_line,
      sym_blank_line,
      sym_heading,
      sym_object_line,
      aux_sym_source_file_repeat1,
  [267] = 5,
    ACTIONS(5), 1,
      anon_sym_LF,
    ACTIONS(7), 1,
      sym_heading_marker,
    ACTIONS(94), 1,
      ts_builtin_sym_end,
    ACTIONS(11), 2,
      anon_sym_DASH,
      anon_sym_PERCENT,
    STATE(9), 5,
      sym__body_line,
      sym_blank_line,
      sym_heading,
      sym_object_line,
      aux_sym_source_file_repeat1,
  [288] = 3,
    ACTIONS(111), 1,
      anon_sym_EQ,
    ACTIONS(113), 1,
      anon_sym_SLASH,
    ACTIONS(109), 7,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      sym_box_ident,
      anon_sym_DQUOTE,
  [304] = 1,
    ACTIONS(116), 9,
      anon_sym_LF,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [316] = 1,
    ACTIONS(118), 9,
      anon_sym_LF,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [328] = 1,
    ACTIONS(120), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [339] = 1,
    ACTIONS(122), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [350] = 1,
    ACTIONS(124), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [361] = 1,
    ACTIONS(126), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [372] = 1,
    ACTIONS(128), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [383] = 1,
    ACTIONS(130), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [394] = 1,
    ACTIONS(132), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [405] = 1,
    ACTIONS(134), 8,
      anon_sym_PERCENT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_RBRACK,
      sym__box_space,
      anon_sym_SLASH,
      sym_box_ident,
      anon_sym_DQUOTE,
  [416] = 2,
    ACTIONS(136), 1,
      anon_sym_DASH,
    ACTIONS(138), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [428] = 2,
    ACTIONS(140), 1,
      anon_sym_DASH,
    ACTIONS(142), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [440] = 2,
    ACTIONS(144), 1,
      anon_sym_DASH,
    ACTIONS(146), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [452] = 2,
    ACTIONS(148), 1,
      anon_sym_DASH,
    ACTIONS(150), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [464] = 2,
    ACTIONS(152), 1,
      anon_sym_DASH,
    ACTIONS(154), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [476] = 2,
    ACTIONS(156), 1,
      anon_sym_DASH,
    ACTIONS(158), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [488] = 2,
    ACTIONS(160), 1,
      anon_sym_DASH,
    ACTIONS(162), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [500] = 2,
    ACTIONS(164), 1,
      anon_sym_DASH,
    ACTIONS(166), 6,
      anon_sym_PERCENT,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
  [512] = 1,
    ACTIONS(168), 5,
      ts_builtin_sym_end,
      anon_sym_LF,
      sym_heading_marker,
      anon_sym_DASH,
      anon_sym_PERCENT,
  [520] = 1,
    ACTIONS(170), 5,
      ts_builtin_sym_end,
      anon_sym_LF,
      sym_heading_marker,
      anon_sym_DASH,
      anon_sym_PERCENT,
  [528] = 1,
    ACTIONS(172), 5,
      ts_builtin_sym_end,
      anon_sym_LF,
      sym_heading_marker,
      anon_sym_DASH,
      anon_sym_PERCENT,
  [536] = 1,
    ACTIONS(174), 5,
      ts_builtin_sym_end,
      anon_sym_LF,
      sym_heading_marker,
      anon_sym_DASH,
      anon_sym_PERCENT,
  [544] = 1,
    ACTIONS(176), 5,
      ts_builtin_sym_end,
      anon_sym_LF,
      sym_heading_marker,
      anon_sym_DASH,
      anon_sym_PERCENT,
  [552] = 1,
    ACTIONS(178), 5,
      ts_builtin_sym_end,
      anon_sym_LF,
      sym_heading_marker,
      anon_sym_DASH,
      anon_sym_PERCENT,
  [560] = 1,
    ACTIONS(180), 5,
      ts_builtin_sym_end,
      anon_sym_LF,
      sym_heading_marker,
      anon_sym_DASH,
      anon_sym_PERCENT,
  [568] = 3,
    ACTIONS(29), 1,
      anon_sym_DQUOTE,
    ACTIONS(182), 1,
      sym_box_bare_value,
    STATE(20), 2,
      sym__box_value,
      sym_box_quoted,
  [579] = 4,
    ACTIONS(184), 1,
      sym_file_path,
    ACTIONS(186), 1,
      sym_markl_purpose,
    ACTIONS(188), 1,
      sym_markl_format,
    STATE(73), 1,
      sym_markl_id,
  [592] = 3,
    ACTIONS(190), 1,
      anon_sym_DQUOTE,
    STATE(40), 1,
      aux_sym_box_quoted_repeat1,
    ACTIONS(192), 2,
      aux_sym_box_quoted_token1,
      sym_box_escape,
  [603] = 3,
    ACTIONS(195), 1,
      anon_sym_DQUOTE,
    STATE(42), 1,
      aux_sym_box_quoted_repeat1,
    ACTIONS(197), 2,
      aux_sym_box_quoted_token1,
      sym_box_escape,
  [614] = 3,
    ACTIONS(199), 1,
      anon_sym_DQUOTE,
    STATE(40), 1,
      aux_sym_box_quoted_repeat1,
    ACTIONS(201), 2,
      aux_sym_box_quoted_token1,
      sym_box_escape,
  [625] = 3,
    ACTIONS(203), 1,
      anon_sym_LF,
    ACTIONS(205), 1,
      sym__heading_comma,
    STATE(44), 1,
      aux_sym_heading_repeat1,
  [635] = 3,
    ACTIONS(207), 1,
      anon_sym_LF,
    ACTIONS(209), 1,
      sym__heading_comma,
    STATE(44), 1,
      aux_sym_heading_repeat1,
  [645] = 3,
    ACTIONS(186), 1,
      sym_markl_purpose,
    ACTIONS(188), 1,
      sym_markl_format,
    STATE(19), 1,
      sym_markl_id,
  [655] = 3,
    ACTIONS(205), 1,
      sym__heading_comma,
    ACTIONS(212), 1,
      anon_sym_LF,
    STATE(43), 1,
      aux_sym_heading_repeat1,
  [665] = 3,
    ACTIONS(186), 1,
      sym_markl_purpose,
    ACTIONS(188), 1,
      sym_markl_format,
    STATE(89), 1,
      sym_markl_id,
  [675] = 3,
    ACTIONS(214), 1,
      anon_sym_LF,
    ACTIONS(216), 1,
      aux_sym_description_token1,
    STATE(76), 1,
      sym_comment_text,
  [685] = 2,
    ACTIONS(218), 1,
      aux_sym_description_token1,
    STATE(69), 1,
      sym_description_text,
  [692] = 1,
    ACTIONS(207), 2,
      anon_sym_LF,
      sym__heading_comma,
  [697] = 2,
    ACTIONS(220), 1,
      aux_sym_description_token1,
    STATE(71), 1,
      sym_reference_id,
  [704] = 2,
    ACTIONS(222), 1,
      anon_sym_LF,
    ACTIONS(224), 1,
      anon_sym_SPACE,
  [711] = 1,
    ACTIONS(226), 2,
      anon_sym_LF,
      anon_sym_SPACE,
  [716] = 2,
    ACTIONS(228), 1,
      anon_sym_LBRACK,
    STATE(52), 1,
      sym_box,
  [723] = 1,
    ACTIONS(230), 2,
      anon_sym_LF,
      anon_sym_SPACE,
  [728] = 2,
    ACTIONS(232), 1,
      anon_sym_LF,
    ACTIONS(234), 1,
      anon_sym_AT,
  [735] = 2,
    ACTIONS(236), 1,
      aux_sym_description_token1,
    STATE(64), 1,
      sym_description,
  [742] = 2,
    ACTIONS(238), 1,
      aux_sym_description_token1,
    STATE(66), 1,
      sym_tag_name,
  [749] = 1,
    ACTIONS(240), 1,
      sym_markl_data,
  [753] = 1,
    ACTIONS(242), 1,
      sym_heading_tag,
  [757] = 1,
    ACTIONS(244), 1,
      anon_sym_SPACE,
  [761] = 1,
    ACTIONS(246), 1,
      anon_sym_SPACE,
  [765] = 1,
    ACTIONS(248), 1,
      anon_sym_LF,
  [769] = 1,
    ACTIONS(250), 1,
      anon_sym_LF,
  [773] = 1,
    ACTIONS(252), 1,
      anon_sym_LF,
  [777] = 1,
    ACTIONS(254), 1,
      anon_sym_LF,
  [781] = 1,
    ACTIONS(256), 1,
      sym_box_ident,
  [785] = 1,
    ACTIONS(258), 1,
      anon_sym_LF,
  [789] = 1,
    ACTIONS(260), 1,
      anon_sym_LF,
  [793] = 1,
    ACTIONS(262), 1,
      anon_sym_LF,
  [797] = 1,
    ACTIONS(264), 1,
      anon_sym_LF,
  [801] = 1,
    ACTIONS(266), 1,
      anon_sym_LF,
  [805] = 1,
    ACTIONS(268), 1,
      anon_sym_LF,
  [809] = 1,
    ACTIONS(270), 1,
      anon_sym_DASH,
  [813] = 1,
    ACTIONS(272), 1,
      sym_markl_data,
  [817] = 1,
    ACTIONS(274), 1,
      anon_sym_LF,
  [821] = 1,
    ACTIONS(276), 1,
      anon_sym_LF,
  [825] = 1,
    ACTIONS(278), 1,
      anon_sym_SPACE,
  [829] = 1,
    ACTIONS(280), 1,
      ts_builtin_sym_end,
  [833] = 1,
    ACTIONS(282), 1,
      sym_markl_format,
  [837] = 1,
    ACTIONS(284), 1,
      anon_sym_DASH,
  [841] = 1,
    ACTIONS(286), 1,
      sym_heading_tag,
  [845] = 1,
    ACTIONS(288), 1,
      sym_box_ident,
  [849] = 1,
    ACTIONS(290), 1,
      anon_sym_SPACE,
  [853] = 1,
    ACTIONS(292), 1,
      sym_type_name,
  [857] = 1,
    ACTIONS(294), 1,
      sym_box_ident,
  [861] = 1,
    ACTIONS(296), 1,
      anon_sym_SPACE,
  [865] = 1,
    ACTIONS(298), 1,
      anon_sym_SPACE,
  [869] = 1,
    ACTIONS(300), 1,
      anon_sym_LF,
  [873] = 1,
    ACTIONS(302), 1,
      anon_sym_SPACE,
  [877] = 1,
    ACTIONS(304), 1,
      sym_box_ident,
};

static const uint32_t ts_small_parse_table_map[] = {
  [SMALL_STATE(2)] = 0,
  [SMALL_STATE(3)] = 36,
  [SMALL_STATE(4)] = 72,
  [SMALL_STATE(5)] = 108,
  [SMALL_STATE(6)] = 140,
  [SMALL_STATE(7)] = 172,
  [SMALL_STATE(8)] = 204,
  [SMALL_STATE(9)] = 225,
  [SMALL_STATE(10)] = 246,
  [SMALL_STATE(11)] = 267,
  [SMALL_STATE(12)] = 288,
  [SMALL_STATE(13)] = 304,
  [SMALL_STATE(14)] = 316,
  [SMALL_STATE(15)] = 328,
  [SMALL_STATE(16)] = 339,
  [SMALL_STATE(17)] = 350,
  [SMALL_STATE(18)] = 361,
  [SMALL_STATE(19)] = 372,
  [SMALL_STATE(20)] = 383,
  [SMALL_STATE(21)] = 394,
  [SMALL_STATE(22)] = 405,
  [SMALL_STATE(23)] = 416,
  [SMALL_STATE(24)] = 428,
  [SMALL_STATE(25)] = 440,
  [SMALL_STATE(26)] = 452,
  [SMALL_STATE(27)] = 464,
  [SMALL_STATE(28)] = 476,
  [SMALL_STATE(29)] = 488,
  [SMALL_STATE(30)] = 500,
  [SMALL_STATE(31)] = 512,
  [SMALL_STATE(32)] = 520,
  [SMALL_STATE(33)] = 528,
  [SMALL_STATE(34)] = 536,
  [SMALL_STATE(35)] = 544,
  [SMALL_STATE(36)] = 552,
  [SMALL_STATE(37)] = 560,
  [SMALL_STATE(38)] = 568,
  [SMALL_STATE(39)] = 579,
  [SMALL_STATE(40)] = 592,
  [SMALL_STATE(41)] = 603,
  [SMALL_STATE(42)] = 614,
  [SMALL_STATE(43)] = 625,
  [SMALL_STATE(44)] = 635,
  [SMALL_STATE(45)] = 645,
  [SMALL_STATE(46)] = 655,
  [SMALL_STATE(47)] = 665,
  [SMALL_STATE(48)] = 675,
  [SMALL_STATE(49)] = 685,
  [SMALL_STATE(50)] = 692,
  [SMALL_STATE(51)] = 697,
  [SMALL_STATE(52)] = 704,
  [SMALL_STATE(53)] = 711,
  [SMALL_STATE(54)] = 716,
  [SMALL_STATE(55)] = 723,
  [SMALL_STATE(56)] = 728,
  [SMALL_STATE(57)] = 735,
  [SMALL_STATE(58)] = 742,
  [SMALL_STATE(59)] = 749,
  [SMALL_STATE(60)] = 753,
  [SMALL_STATE(61)] = 757,
  [SMALL_STATE(62)] = 761,
  [SMALL_STATE(63)] = 765,
  [SMALL_STATE(64)] = 769,
  [SMALL_STATE(65)] = 773,
  [SMALL_STATE(66)] = 777,
  [SMALL_STATE(67)] = 781,
  [SMALL_STATE(68)] = 785,
  [SMALL_STATE(69)] = 789,
  [SMALL_STATE(70)] = 793,
  [SMALL_STATE(71)] = 797,
  [SMALL_STATE(72)] = 801,
  [SMALL_STATE(73)] = 805,
  [SMALL_STATE(74)] = 809,
  [SMALL_STATE(75)] = 813,
  [SMALL_STATE(76)] = 817,
  [SMALL_STATE(77)] = 821,
  [SMALL_STATE(78)] = 825,
  [SMALL_STATE(79)] = 829,
  [SMALL_STATE(80)] = 833,
  [SMALL_STATE(81)] = 837,
  [SMALL_STATE(82)] = 841,
  [SMALL_STATE(83)] = 845,
  [SMALL_STATE(84)] = 849,
  [SMALL_STATE(85)] = 853,
  [SMALL_STATE(86)] = 857,
  [SMALL_STATE(87)] = 861,
  [SMALL_STATE(88)] = 865,
  [SMALL_STATE(89)] = 869,
  [SMALL_STATE(90)] = 873,
  [SMALL_STATE(91)] = 877,
};

static const TSParseActionEntry ts_parse_actions[] = {
  [0] = {.entry = {.count = 0, .reusable = false}},
  [1] = {.entry = {.count = 1, .reusable = false}}, RECOVER(),
  [3] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_source_file, 0, 0, 0),
  [5] = {.entry = {.count = 1, .reusable = true}}, SHIFT(33),
  [7] = {.entry = {.count = 1, .reusable = true}}, SHIFT(87),
  [9] = {.entry = {.count = 1, .reusable = false}}, SHIFT(61),
  [11] = {.entry = {.count = 1, .reusable = true}}, SHIFT(61),
  [13] = {.entry = {.count = 1, .reusable = true}}, SHIFT(77),
  [15] = {.entry = {.count = 1, .reusable = true}}, SHIFT(83),
  [17] = {.entry = {.count = 1, .reusable = true}}, SHIFT(67),
  [19] = {.entry = {.count = 1, .reusable = true}}, SHIFT(45),
  [21] = {.entry = {.count = 1, .reusable = true}}, SHIFT(53),
  [23] = {.entry = {.count = 1, .reusable = true}}, SHIFT(3),
  [25] = {.entry = {.count = 1, .reusable = true}}, SHIFT(91),
  [27] = {.entry = {.count = 1, .reusable = true}}, SHIFT(12),
  [29] = {.entry = {.count = 1, .reusable = true}}, SHIFT(41),
  [31] = {.entry = {.count = 1, .reusable = true}}, SHIFT(55),
  [33] = {.entry = {.count = 1, .reusable = true}}, SHIFT(4),
  [35] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0), SHIFT_REPEAT(83),
  [38] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0), SHIFT_REPEAT(67),
  [41] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0), SHIFT_REPEAT(45),
  [44] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0),
  [46] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0), SHIFT_REPEAT(4),
  [49] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0), SHIFT_REPEAT(91),
  [52] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0), SHIFT_REPEAT(12),
  [55] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_repeat1, 2, 0, 0), SHIFT_REPEAT(41),
  [58] = {.entry = {.count = 1, .reusable = false}}, SHIFT(62),
  [60] = {.entry = {.count = 1, .reusable = true}}, SHIFT(48),
  [62] = {.entry = {.count = 1, .reusable = true}}, SHIFT(32),
  [64] = {.entry = {.count = 1, .reusable = true}}, SHIFT(78),
  [66] = {.entry = {.count = 1, .reusable = true}}, SHIFT(84),
  [68] = {.entry = {.count = 1, .reusable = true}}, SHIFT(88),
  [70] = {.entry = {.count = 1, .reusable = true}}, SHIFT(90),
  [72] = {.entry = {.count = 1, .reusable = true}}, SHIFT(35),
  [74] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(62),
  [77] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(48),
  [80] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0),
  [82] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(78),
  [85] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(84),
  [88] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(88),
  [91] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(90),
  [94] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_source_file, 1, 0, 0),
  [96] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_source_file, 2, 0, 0),
  [98] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_source_file_repeat1, 2, 0, 0),
  [100] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_source_file_repeat1, 2, 0, 0), SHIFT_REPEAT(33),
  [103] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_source_file_repeat1, 2, 0, 0), SHIFT_REPEAT(87),
  [106] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_source_file_repeat1, 2, 0, 0), SHIFT_REPEAT(61),
  [109] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_tag, 1, 0, 0),
  [111] = {.entry = {.count = 1, .reusable = true}}, SHIFT(38),
  [113] = {.entry = {.count = 2, .reusable = true}}, REDUCE(sym_box_tag, 1, 0, 0), SHIFT(86),
  [116] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_markl_id, 3, 0, 12),
  [118] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_markl_id, 4, 0, 13),
  [120] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_type, 2, 0, 3),
  [122] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_computed_tag, 2, 0, 3),
  [124] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_object_id, 2, 0, 3),
  [126] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_quoted, 2, 0, 0),
  [128] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_blob, 2, 0, 0),
  [130] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_field, 3, 0, 5),
  [132] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_object_id, 3, 0, 6),
  [134] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box_quoted, 3, 0, 0),
  [136] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_type_line, 4, 0, 11),
  [138] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_type_line, 4, 0, 11),
  [140] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_type_line, 6, 0, 14),
  [142] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_type_line, 6, 0, 14),
  [144] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_comment_line, 2, 0, 0),
  [146] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment_line, 2, 0, 0),
  [148] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_comment_line, 3, 0, 4),
  [150] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment_line, 3, 0, 4),
  [152] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_tag_line, 4, 0, 8),
  [154] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_tag_line, 4, 0, 8),
  [156] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_description_line, 4, 0, 9),
  [158] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_description_line, 4, 0, 9),
  [160] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_ref_line, 4, 0, 10),
  [162] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_ref_line, 4, 0, 10),
  [164] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_blob_line, 4, 0, 10),
  [166] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_blob_line, 4, 0, 10),
  [168] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_heading, 4, 0, 1),
  [170] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_metadata, 4, 0, 0),
  [172] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_blank_line, 1, 0, 0),
  [174] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_heading, 5, 0, 1),
  [176] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_metadata, 3, 0, 0),
  [178] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_object_line, 4, 0, 2),
  [180] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_object_line, 6, 0, 7),
  [182] = {.entry = {.count = 1, .reusable = true}}, SHIFT(20),
  [184] = {.entry = {.count = 1, .reusable = true}}, SHIFT(73),
  [186] = {.entry = {.count = 1, .reusable = true}}, SHIFT(80),
  [188] = {.entry = {.count = 1, .reusable = false}}, SHIFT(81),
  [190] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_box_quoted_repeat1, 2, 0, 0),
  [192] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_box_quoted_repeat1, 2, 0, 0), SHIFT_REPEAT(40),
  [195] = {.entry = {.count = 1, .reusable = true}}, SHIFT(18),
  [197] = {.entry = {.count = 1, .reusable = true}}, SHIFT(42),
  [199] = {.entry = {.count = 1, .reusable = true}}, SHIFT(22),
  [201] = {.entry = {.count = 1, .reusable = true}}, SHIFT(40),
  [203] = {.entry = {.count = 1, .reusable = true}}, SHIFT(34),
  [205] = {.entry = {.count = 1, .reusable = true}}, SHIFT(82),
  [207] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_heading_repeat1, 2, 0, 0),
  [209] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_heading_repeat1, 2, 0, 0), SHIFT_REPEAT(82),
  [212] = {.entry = {.count = 1, .reusable = true}}, SHIFT(31),
  [214] = {.entry = {.count = 1, .reusable = true}}, SHIFT(25),
  [216] = {.entry = {.count = 1, .reusable = true}}, SHIFT(72),
  [218] = {.entry = {.count = 1, .reusable = true}}, SHIFT(68),
  [220] = {.entry = {.count = 1, .reusable = true}}, SHIFT(70),
  [222] = {.entry = {.count = 1, .reusable = true}}, SHIFT(36),
  [224] = {.entry = {.count = 1, .reusable = true}}, SHIFT(57),
  [226] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box, 2, 0, 0),
  [228] = {.entry = {.count = 1, .reusable = true}}, SHIFT(2),
  [230] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_box, 3, 0, 0),
  [232] = {.entry = {.count = 1, .reusable = true}}, SHIFT(23),
  [234] = {.entry = {.count = 1, .reusable = true}}, SHIFT(47),
  [236] = {.entry = {.count = 1, .reusable = true}}, SHIFT(63),
  [238] = {.entry = {.count = 1, .reusable = true}}, SHIFT(65),
  [240] = {.entry = {.count = 1, .reusable = true}}, SHIFT(14),
  [242] = {.entry = {.count = 1, .reusable = true}}, SHIFT(46),
  [244] = {.entry = {.count = 1, .reusable = true}}, SHIFT(54),
  [246] = {.entry = {.count = 1, .reusable = true}}, SHIFT(58),
  [248] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_description, 1, 0, 0),
  [250] = {.entry = {.count = 1, .reusable = true}}, SHIFT(37),
  [252] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_tag_name, 1, 0, 0),
  [254] = {.entry = {.count = 1, .reusable = true}}, SHIFT(27),
  [256] = {.entry = {.count = 1, .reusable = true}}, SHIFT(15),
  [258] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_description_text, 1, 0, 0),
  [260] = {.entry = {.count = 1, .reusable = true}}, SHIFT(28),
  [262] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_reference_id, 1, 0, 0),
  [264] = {.entry = {.count = 1, .reusable = true}}, SHIFT(29),
  [266] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment_text, 1, 0, 0),
  [268] = {.entry = {.count = 1, .reusable = true}}, SHIFT(30),
  [270] = {.entry = {.count = 1, .reusable = true}}, SHIFT(59),
  [272] = {.entry = {.count = 1, .reusable = true}}, SHIFT(13),
  [274] = {.entry = {.count = 1, .reusable = true}}, SHIFT(26),
  [276] = {.entry = {.count = 1, .reusable = true}}, SHIFT(6),
  [278] = {.entry = {.count = 1, .reusable = true}}, SHIFT(49),
  [280] = {.entry = {.count = 1, .reusable = true}},  ACCEPT_INPUT(),
  [282] = {.entry = {.count = 1, .reusable = true}}, SHIFT(74),
  [284] = {.entry = {.count = 1, .reusable = true}}, SHIFT(75),
  [286] = {.entry = {.count = 1, .reusable = true}}, SHIFT(50),
  [288] = {.entry = {.count = 1, .reusable = true}}, SHIFT(16),
  [290] = {.entry = {.count = 1, .reusable = true}}, SHIFT(51),
  [292] = {.entry = {.count = 1, .reusable = true}}, SHIFT(56),
  [294] = {.entry = {.count = 1, .reusable = true}}, SHIFT(21),
  [296] = {.entry = {.count = 1, .reusable = true}}, SHIFT(60),
  [298] = {.entry = {.count = 1, .reusable = true}}, SHIFT(85),
  [300] = {.entry = {.count = 1, .reusable = true}}, SHIFT(24),
  [302] = {.entry = {.count = 1, .reusable = true}}, SHIFT(39),
  [304] = {.entry = {.count = 1, .reusable = true}}, SHIFT(17),
};

#ifdef __cplusplus
extern "C" {
#endif
#ifdef TREE_SITTER_HIDE_SYMBOLS
#define TS_PUBLIC
#elif defined(_WIN32)
#define TS_PUBLIC __declspec(dllexport)
#else
#define TS_PUBLIC __attribute__((visibility("default")))
#endif

TS_PUBLIC const TSLanguage *tree_sitter_dodder_organize(void) {
  static const TSLanguage language = {
    .version = LANGUAGE_VERSION,
    .symbol_count = SYMBOL_COUNT,
    .alias_count = ALIAS_COUNT,
    .token_count = TOKEN_COUNT,
    .external_token_count = EXTERNAL_TOKEN_COUNT,
    .state_count = STATE_COUNT,
    .large_state_count = LARGE_STATE_COUNT,
    .production_id_count = PRODUCTION_ID_COUNT,
    .field_count = FIELD_COUNT,
    .max_alias_sequence_length = MAX_ALIAS_SEQUENCE_LENGTH,
    .parse_table = &ts_parse_table[0][0],
    .small_parse_table = ts_small_parse_table,
    .small_parse_table_map = ts_small_parse_table_map,
    .parse_actions = ts_parse_actions,
    .symbol_names = ts_symbol_names,
    .field_names = ts_field_names,
    .field_map_slices = ts_field_map_slices,
    .field_map_entries = ts_field_map_entries,
    .symbol_metadata = ts_symbol_metadata,
    .public_symbol_map = ts_symbol_map,
    .alias_map = ts_non_terminal_alias_map,
    .alias_sequences = &ts_alias_sequences[0][0],
    .lex_modes = ts_lex_modes,
    .lex_fn = ts_lex,
    .primary_state_ids = ts_primary_state_ids,
  };
  return &language;
}
#ifdef __cplusplus
}
#endif
