#include "tree_sitter/parser.h"

#if defined(__GNUC__) || defined(__clang__)
#pragma GCC diagnostic ignored "-Wmissing-field-initializers"
#endif

#define LANGUAGE_VERSION 14
#define STATE_COUNT 49
#define LARGE_STATE_COUNT 2
#define SYMBOL_COUNT 32
#define ALIAS_COUNT 0
#define TOKEN_COUNT 17
#define EXTERNAL_TOKEN_COUNT 0
#define FIELD_COUNT 7
#define MAX_ALIAS_SEQUENCE_LENGTH 6
#define PRODUCTION_ID_COUNT 9

enum ts_symbol_identifiers {
  sym_body = 1,
  anon_sym_DASH_DASH_DASH = 2,
  anon_sym_LF = 3,
  anon_sym_POUND = 4,
  anon_sym_SPACE = 5,
  aux_sym_description_text_token1 = 6,
  anon_sym_DASH = 7,
  anon_sym_LT = 8,
  anon_sym_BANG = 9,
  anon_sym_AT = 10,
  sym_type_name = 11,
  sym_file_path = 12,
  anon_sym_PERCENT = 13,
  sym_markl_purpose = 14,
  sym_markl_format = 15,
  sym_markl_data = 16,
  sym_source_file = 17,
  sym_metadata = 18,
  sym__metadata_line = 19,
  sym_description_line = 20,
  sym_description_text = 21,
  sym_tag_line = 22,
  sym_tag_name = 23,
  sym_ref_line = 24,
  sym_reference_id = 25,
  sym_type_line = 26,
  sym_blob_line = 27,
  sym_comment_line = 28,
  sym_comment_text = 29,
  sym_markl_id = 30,
  aux_sym_metadata_repeat1 = 31,
};

static const char * const ts_symbol_names[] = {
  [ts_builtin_sym_end] = "end",
  [sym_body] = "body",
  [anon_sym_DASH_DASH_DASH] = "---",
  [anon_sym_LF] = "\n",
  [anon_sym_POUND] = "#",
  [anon_sym_SPACE] = " ",
  [aux_sym_description_text_token1] = "description_text_token1",
  [anon_sym_DASH] = "-",
  [anon_sym_LT] = "<",
  [anon_sym_BANG] = "!",
  [anon_sym_AT] = "@",
  [sym_type_name] = "type_name",
  [sym_file_path] = "file_path",
  [anon_sym_PERCENT] = "%",
  [sym_markl_purpose] = "markl_purpose",
  [sym_markl_format] = "markl_format",
  [sym_markl_data] = "markl_data",
  [sym_source_file] = "source_file",
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
  [sym_markl_id] = "markl_id",
  [aux_sym_metadata_repeat1] = "metadata_repeat1",
};

static const TSSymbol ts_symbol_map[] = {
  [ts_builtin_sym_end] = ts_builtin_sym_end,
  [sym_body] = sym_body,
  [anon_sym_DASH_DASH_DASH] = anon_sym_DASH_DASH_DASH,
  [anon_sym_LF] = anon_sym_LF,
  [anon_sym_POUND] = anon_sym_POUND,
  [anon_sym_SPACE] = anon_sym_SPACE,
  [aux_sym_description_text_token1] = aux_sym_description_text_token1,
  [anon_sym_DASH] = anon_sym_DASH,
  [anon_sym_LT] = anon_sym_LT,
  [anon_sym_BANG] = anon_sym_BANG,
  [anon_sym_AT] = anon_sym_AT,
  [sym_type_name] = sym_type_name,
  [sym_file_path] = sym_file_path,
  [anon_sym_PERCENT] = anon_sym_PERCENT,
  [sym_markl_purpose] = sym_markl_purpose,
  [sym_markl_format] = sym_markl_format,
  [sym_markl_data] = sym_markl_data,
  [sym_source_file] = sym_source_file,
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
  [sym_markl_id] = sym_markl_id,
  [aux_sym_metadata_repeat1] = aux_sym_metadata_repeat1,
};

static const TSSymbolMetadata ts_symbol_metadata[] = {
  [ts_builtin_sym_end] = {
    .visible = false,
    .named = true,
  },
  [sym_body] = {
    .visible = true,
    .named = true,
  },
  [anon_sym_DASH_DASH_DASH] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_LF] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_POUND] = {
    .visible = true,
    .named = false,
  },
  [anon_sym_SPACE] = {
    .visible = true,
    .named = false,
  },
  [aux_sym_description_text_token1] = {
    .visible = false,
    .named = false,
  },
  [anon_sym_DASH] = {
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
  [anon_sym_PERCENT] = {
    .visible = true,
    .named = false,
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
  [sym_markl_id] = {
    .visible = true,
    .named = true,
  },
  [aux_sym_metadata_repeat1] = {
    .visible = false,
    .named = false,
  },
};

enum ts_field_identifiers {
  field_data = 1,
  field_format = 2,
  field_lock = 3,
  field_name = 4,
  field_ref = 5,
  field_text = 6,
  field_type = 7,
};

static const char * const ts_field_names[] = {
  [0] = NULL,
  [field_data] = "data",
  [field_format] = "format",
  [field_lock] = "lock",
  [field_name] = "name",
  [field_ref] = "ref",
  [field_text] = "text",
  [field_type] = "type",
};

static const TSFieldMapSlice ts_field_map_slices[PRODUCTION_ID_COUNT] = {
  [1] = {.index = 0, .length = 1},
  [2] = {.index = 1, .length = 1},
  [3] = {.index = 2, .length = 1},
  [4] = {.index = 3, .length = 1},
  [5] = {.index = 4, .length = 1},
  [6] = {.index = 5, .length = 2},
  [7] = {.index = 7, .length = 2},
  [8] = {.index = 9, .length = 2},
};

static const TSFieldMapEntry ts_field_map_entries[] = {
  [0] =
    {field_text, 1},
  [1] =
    {field_text, 2},
  [2] =
    {field_name, 2},
  [3] =
    {field_ref, 2},
  [4] =
    {field_type, 2},
  [5] =
    {field_data, 2},
    {field_format, 0},
  [7] =
    {field_lock, 4},
    {field_type, 2},
  [9] =
    {field_data, 3},
    {field_format, 1},
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
};

static bool ts_lex(TSLexer *lexer, TSStateId state) {
  START_LEXER();
  eof = lexer->eof(lexer);
  switch (state) {
    case 0:
      if (eof) ADVANCE(13);
      ADVANCE_MAP(
        '\n', 17,
        ' ', 19,
        '!', 23,
        '#', 18,
        '%', 27,
        '-', 21,
        '<', 22,
        '@', 24,
        '_', 30,
      );
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(29);
      END_STATE();
    case 1:
      if (lookahead == '\n') ADVANCE(14);
      END_STATE();
    case 2:
      if (lookahead == '\n') ADVANCE(16);
      if (lookahead == '@') ADVANCE(24);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(30);
      END_STATE();
    case 3:
      if (lookahead == '\n') ADVANCE(16);
      if (lookahead != 0) ADVANCE(20);
      END_STATE();
    case 4:
      if (lookahead == '-') ADVANCE(15);
      END_STATE();
    case 5:
      if (lookahead == '.') ADVANCE(6);
      if (lookahead == '@') ADVANCE(28);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(5);
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(7);
      END_STATE();
    case 6:
      if (lookahead == '.') ADVANCE(6);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(26);
      END_STATE();
    case 7:
      if (lookahead == '.') ADVANCE(6);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(7);
      END_STATE();
    case 8:
      if (lookahead == '@') ADVANCE(28);
      if (lookahead == '-' ||
          ('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(8);
      END_STATE();
    case 9:
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(33);
      END_STATE();
    case 10:
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(32);
      END_STATE();
    case 11:
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(25);
      END_STATE();
    case 12:
      if (eof) ADVANCE(13);
      if (lookahead == '\n') ADVANCE(1);
      if (lookahead == '.') ADVANCE(6);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(31);
      if (lookahead != 0 &&
          lookahead != '@') ADVANCE(7);
      END_STATE();
    case 13:
      ACCEPT_TOKEN(ts_builtin_sym_end);
      END_STATE();
    case 14:
      ACCEPT_TOKEN(sym_body);
      if (lookahead != 0) ADVANCE(14);
      END_STATE();
    case 15:
      ACCEPT_TOKEN(anon_sym_DASH_DASH_DASH);
      END_STATE();
    case 16:
      ACCEPT_TOKEN(anon_sym_LF);
      END_STATE();
    case 17:
      ACCEPT_TOKEN(anon_sym_LF);
      if (lookahead == '\n') ADVANCE(14);
      END_STATE();
    case 18:
      ACCEPT_TOKEN(anon_sym_POUND);
      END_STATE();
    case 19:
      ACCEPT_TOKEN(anon_sym_SPACE);
      END_STATE();
    case 20:
      ACCEPT_TOKEN(aux_sym_description_text_token1);
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(20);
      END_STATE();
    case 21:
      ACCEPT_TOKEN(anon_sym_DASH);
      if (lookahead == '-') ADVANCE(4);
      END_STATE();
    case 22:
      ACCEPT_TOKEN(anon_sym_LT);
      END_STATE();
    case 23:
      ACCEPT_TOKEN(anon_sym_BANG);
      END_STATE();
    case 24:
      ACCEPT_TOKEN(anon_sym_AT);
      END_STATE();
    case 25:
      ACCEPT_TOKEN(sym_type_name);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(25);
      END_STATE();
    case 26:
      ACCEPT_TOKEN(sym_file_path);
      if (lookahead == '.') ADVANCE(6);
      if (lookahead != 0 &&
          lookahead != '\n' &&
          lookahead != '@') ADVANCE(26);
      END_STATE();
    case 27:
      ACCEPT_TOKEN(anon_sym_PERCENT);
      END_STATE();
    case 28:
      ACCEPT_TOKEN(sym_markl_purpose);
      END_STATE();
    case 29:
      ACCEPT_TOKEN(sym_markl_format);
      if (lookahead == '-') ADVANCE(8);
      if (lookahead == '@') ADVANCE(28);
      if (lookahead == '_') ADVANCE(30);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(29);
      END_STATE();
    case 30:
      ACCEPT_TOKEN(sym_markl_format);
      if (lookahead == '-') ADVANCE(8);
      if (lookahead == '@') ADVANCE(28);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(30);
      END_STATE();
    case 31:
      ACCEPT_TOKEN(sym_markl_format);
      if (lookahead == '-') ADVANCE(5);
      if (lookahead == '.') ADVANCE(6);
      if (lookahead == '@') ADVANCE(28);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(31);
      if (lookahead != 0 &&
          lookahead != '\n') ADVANCE(7);
      END_STATE();
    case 32:
      ACCEPT_TOKEN(sym_markl_format);
      if (('0' <= lookahead && lookahead <= '9') ||
          lookahead == '_' ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(32);
      END_STATE();
    case 33:
      ACCEPT_TOKEN(sym_markl_data);
      if (('0' <= lookahead && lookahead <= '9') ||
          ('a' <= lookahead && lookahead <= 'z')) ADVANCE(33);
      END_STATE();
    default:
      return false;
  }
}

static const TSLexMode ts_lex_modes[STATE_COUNT] = {
  [0] = {.lex_state = 0},
  [1] = {.lex_state = 0},
  [2] = {.lex_state = 0},
  [3] = {.lex_state = 0},
  [4] = {.lex_state = 0},
  [5] = {.lex_state = 0},
  [6] = {.lex_state = 0},
  [7] = {.lex_state = 0},
  [8] = {.lex_state = 0},
  [9] = {.lex_state = 0},
  [10] = {.lex_state = 0},
  [11] = {.lex_state = 0},
  [12] = {.lex_state = 0},
  [13] = {.lex_state = 12},
  [14] = {.lex_state = 3},
  [15] = {.lex_state = 2},
  [16] = {.lex_state = 12},
  [17] = {.lex_state = 12},
  [18] = {.lex_state = 2},
  [19] = {.lex_state = 12},
  [20] = {.lex_state = 3},
  [21] = {.lex_state = 3},
  [22] = {.lex_state = 3},
  [23] = {.lex_state = 11},
  [24] = {.lex_state = 0},
  [25] = {.lex_state = 0},
  [26] = {.lex_state = 3},
  [27] = {.lex_state = 3},
  [28] = {.lex_state = 0},
  [29] = {.lex_state = 3},
  [30] = {.lex_state = 3},
  [31] = {.lex_state = 3},
  [32] = {.lex_state = 3},
  [33] = {.lex_state = 3},
  [34] = {.lex_state = 3},
  [35] = {.lex_state = 3},
  [36] = {.lex_state = 3},
  [37] = {.lex_state = 10},
  [38] = {.lex_state = 0},
  [39] = {.lex_state = 0},
  [40] = {.lex_state = 0},
  [41] = {.lex_state = 0},
  [42] = {.lex_state = 0},
  [43] = {.lex_state = 0},
  [44] = {.lex_state = 9},
  [45] = {.lex_state = 3},
  [46] = {.lex_state = 9},
  [47] = {.lex_state = 3},
  [48] = {.lex_state = 3},
};

static const uint16_t ts_parse_table[LARGE_STATE_COUNT][SYMBOL_COUNT] = {
  [0] = {
    [ts_builtin_sym_end] = ACTIONS(1),
    [sym_body] = ACTIONS(1),
    [anon_sym_DASH_DASH_DASH] = ACTIONS(1),
    [anon_sym_LF] = ACTIONS(1),
    [anon_sym_POUND] = ACTIONS(1),
    [anon_sym_SPACE] = ACTIONS(1),
    [anon_sym_DASH] = ACTIONS(1),
    [anon_sym_LT] = ACTIONS(1),
    [anon_sym_BANG] = ACTIONS(1),
    [anon_sym_AT] = ACTIONS(1),
    [anon_sym_PERCENT] = ACTIONS(1),
    [sym_markl_purpose] = ACTIONS(1),
    [sym_markl_format] = ACTIONS(1),
    [sym_markl_data] = ACTIONS(1),
  },
  [1] = {
    [sym_source_file] = STATE(28),
    [sym_metadata] = STATE(17),
    [anon_sym_DASH_DASH_DASH] = ACTIONS(3),
  },
};

static const uint16_t ts_small_parse_table[] = {
  [0] = 8,
    ACTIONS(5), 1,
      anon_sym_DASH_DASH_DASH,
    ACTIONS(7), 1,
      anon_sym_POUND,
    ACTIONS(10), 1,
      anon_sym_DASH,
    ACTIONS(13), 1,
      anon_sym_LT,
    ACTIONS(16), 1,
      anon_sym_BANG,
    ACTIONS(19), 1,
      anon_sym_AT,
    ACTIONS(22), 1,
      anon_sym_PERCENT,
    STATE(2), 8,
      sym__metadata_line,
      sym_description_line,
      sym_tag_line,
      sym_ref_line,
      sym_type_line,
      sym_blob_line,
      sym_comment_line,
      aux_sym_metadata_repeat1,
  [32] = 8,
    ACTIONS(25), 1,
      anon_sym_DASH_DASH_DASH,
    ACTIONS(27), 1,
      anon_sym_POUND,
    ACTIONS(29), 1,
      anon_sym_DASH,
    ACTIONS(31), 1,
      anon_sym_LT,
    ACTIONS(33), 1,
      anon_sym_BANG,
    ACTIONS(35), 1,
      anon_sym_AT,
    ACTIONS(37), 1,
      anon_sym_PERCENT,
    STATE(4), 8,
      sym__metadata_line,
      sym_description_line,
      sym_tag_line,
      sym_ref_line,
      sym_type_line,
      sym_blob_line,
      sym_comment_line,
      aux_sym_metadata_repeat1,
  [64] = 8,
    ACTIONS(27), 1,
      anon_sym_POUND,
    ACTIONS(29), 1,
      anon_sym_DASH,
    ACTIONS(31), 1,
      anon_sym_LT,
    ACTIONS(33), 1,
      anon_sym_BANG,
    ACTIONS(35), 1,
      anon_sym_AT,
    ACTIONS(37), 1,
      anon_sym_PERCENT,
    ACTIONS(39), 1,
      anon_sym_DASH_DASH_DASH,
    STATE(2), 8,
      sym__metadata_line,
      sym_description_line,
      sym_tag_line,
      sym_ref_line,
      sym_type_line,
      sym_blob_line,
      sym_comment_line,
      aux_sym_metadata_repeat1,
  [96] = 2,
    ACTIONS(43), 1,
      anon_sym_DASH,
    ACTIONS(41), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [108] = 2,
    ACTIONS(47), 1,
      anon_sym_DASH,
    ACTIONS(45), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [120] = 2,
    ACTIONS(51), 1,
      anon_sym_DASH,
    ACTIONS(49), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [132] = 2,
    ACTIONS(55), 1,
      anon_sym_DASH,
    ACTIONS(53), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [144] = 2,
    ACTIONS(59), 1,
      anon_sym_DASH,
    ACTIONS(57), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [156] = 2,
    ACTIONS(63), 1,
      anon_sym_DASH,
    ACTIONS(61), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [168] = 2,
    ACTIONS(67), 1,
      anon_sym_DASH,
    ACTIONS(65), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [180] = 2,
    ACTIONS(71), 1,
      anon_sym_DASH,
    ACTIONS(69), 6,
      anon_sym_DASH_DASH_DASH,
      anon_sym_POUND,
      anon_sym_LT,
      anon_sym_BANG,
      anon_sym_AT,
      anon_sym_PERCENT,
  [192] = 4,
    ACTIONS(73), 1,
      sym_file_path,
    ACTIONS(75), 1,
      sym_markl_purpose,
    ACTIONS(77), 1,
      sym_markl_format,
    STATE(36), 1,
      sym_markl_id,
  [205] = 3,
    ACTIONS(79), 1,
      anon_sym_LF,
    ACTIONS(81), 1,
      aux_sym_description_text_token1,
    STATE(27), 1,
      sym_comment_text,
  [215] = 3,
    ACTIONS(75), 1,
      sym_markl_purpose,
    ACTIONS(77), 1,
      sym_markl_format,
    STATE(45), 1,
      sym_markl_id,
  [225] = 1,
    ACTIONS(83), 2,
      ts_builtin_sym_end,
      sym_body,
  [230] = 2,
    ACTIONS(85), 1,
      ts_builtin_sym_end,
    ACTIONS(87), 1,
      sym_body,
  [237] = 2,
    ACTIONS(89), 1,
      anon_sym_LF,
    ACTIONS(91), 1,
      anon_sym_AT,
  [244] = 1,
    ACTIONS(93), 2,
      ts_builtin_sym_end,
      sym_body,
  [249] = 2,
    ACTIONS(95), 1,
      aux_sym_description_text_token1,
    STATE(31), 1,
      sym_description_text,
  [256] = 2,
    ACTIONS(97), 1,
      aux_sym_description_text_token1,
    STATE(33), 1,
      sym_tag_name,
  [263] = 2,
    ACTIONS(99), 1,
      aux_sym_description_text_token1,
    STATE(35), 1,
      sym_reference_id,
  [270] = 1,
    ACTIONS(101), 1,
      sym_type_name,
  [274] = 1,
    ACTIONS(103), 1,
      anon_sym_SPACE,
  [278] = 1,
    ACTIONS(105), 1,
      anon_sym_SPACE,
  [282] = 1,
    ACTIONS(107), 1,
      anon_sym_LF,
  [286] = 1,
    ACTIONS(109), 1,
      anon_sym_LF,
  [290] = 1,
    ACTIONS(111), 1,
      ts_builtin_sym_end,
  [294] = 1,
    ACTIONS(113), 1,
      anon_sym_LF,
  [298] = 1,
    ACTIONS(115), 1,
      anon_sym_LF,
  [302] = 1,
    ACTIONS(117), 1,
      anon_sym_LF,
  [306] = 1,
    ACTIONS(119), 1,
      anon_sym_LF,
  [310] = 1,
    ACTIONS(121), 1,
      anon_sym_LF,
  [314] = 1,
    ACTIONS(123), 1,
      anon_sym_LF,
  [318] = 1,
    ACTIONS(125), 1,
      anon_sym_LF,
  [322] = 1,
    ACTIONS(127), 1,
      anon_sym_LF,
  [326] = 1,
    ACTIONS(129), 1,
      sym_markl_format,
  [330] = 1,
    ACTIONS(131), 1,
      anon_sym_SPACE,
  [334] = 1,
    ACTIONS(133), 1,
      ts_builtin_sym_end,
  [338] = 1,
    ACTIONS(135), 1,
      anon_sym_DASH,
  [342] = 1,
    ACTIONS(137), 1,
      anon_sym_SPACE,
  [346] = 1,
    ACTIONS(139), 1,
      anon_sym_SPACE,
  [350] = 1,
    ACTIONS(141), 1,
      anon_sym_DASH,
  [354] = 1,
    ACTIONS(143), 1,
      sym_markl_data,
  [358] = 1,
    ACTIONS(145), 1,
      anon_sym_LF,
  [362] = 1,
    ACTIONS(147), 1,
      sym_markl_data,
  [366] = 1,
    ACTIONS(149), 1,
      anon_sym_LF,
  [370] = 1,
    ACTIONS(151), 1,
      anon_sym_LF,
};

static const uint32_t ts_small_parse_table_map[] = {
  [SMALL_STATE(2)] = 0,
  [SMALL_STATE(3)] = 32,
  [SMALL_STATE(4)] = 64,
  [SMALL_STATE(5)] = 96,
  [SMALL_STATE(6)] = 108,
  [SMALL_STATE(7)] = 120,
  [SMALL_STATE(8)] = 132,
  [SMALL_STATE(9)] = 144,
  [SMALL_STATE(10)] = 156,
  [SMALL_STATE(11)] = 168,
  [SMALL_STATE(12)] = 180,
  [SMALL_STATE(13)] = 192,
  [SMALL_STATE(14)] = 205,
  [SMALL_STATE(15)] = 215,
  [SMALL_STATE(16)] = 225,
  [SMALL_STATE(17)] = 230,
  [SMALL_STATE(18)] = 237,
  [SMALL_STATE(19)] = 244,
  [SMALL_STATE(20)] = 249,
  [SMALL_STATE(21)] = 256,
  [SMALL_STATE(22)] = 263,
  [SMALL_STATE(23)] = 270,
  [SMALL_STATE(24)] = 274,
  [SMALL_STATE(25)] = 278,
  [SMALL_STATE(26)] = 282,
  [SMALL_STATE(27)] = 286,
  [SMALL_STATE(28)] = 290,
  [SMALL_STATE(29)] = 294,
  [SMALL_STATE(30)] = 298,
  [SMALL_STATE(31)] = 302,
  [SMALL_STATE(32)] = 306,
  [SMALL_STATE(33)] = 310,
  [SMALL_STATE(34)] = 314,
  [SMALL_STATE(35)] = 318,
  [SMALL_STATE(36)] = 322,
  [SMALL_STATE(37)] = 326,
  [SMALL_STATE(38)] = 330,
  [SMALL_STATE(39)] = 334,
  [SMALL_STATE(40)] = 338,
  [SMALL_STATE(41)] = 342,
  [SMALL_STATE(42)] = 346,
  [SMALL_STATE(43)] = 350,
  [SMALL_STATE(44)] = 354,
  [SMALL_STATE(45)] = 358,
  [SMALL_STATE(46)] = 362,
  [SMALL_STATE(47)] = 366,
  [SMALL_STATE(48)] = 370,
};

static const TSParseActionEntry ts_parse_actions[] = {
  [0] = {.entry = {.count = 0, .reusable = false}},
  [1] = {.entry = {.count = 1, .reusable = false}}, RECOVER(),
  [3] = {.entry = {.count = 1, .reusable = true}}, SHIFT(29),
  [5] = {.entry = {.count = 1, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0),
  [7] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(24),
  [10] = {.entry = {.count = 2, .reusable = false}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(25),
  [13] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(38),
  [16] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(42),
  [19] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(41),
  [22] = {.entry = {.count = 2, .reusable = true}}, REDUCE(aux_sym_metadata_repeat1, 2, 0, 0), SHIFT_REPEAT(14),
  [25] = {.entry = {.count = 1, .reusable = true}}, SHIFT(19),
  [27] = {.entry = {.count = 1, .reusable = true}}, SHIFT(24),
  [29] = {.entry = {.count = 1, .reusable = false}}, SHIFT(25),
  [31] = {.entry = {.count = 1, .reusable = true}}, SHIFT(38),
  [33] = {.entry = {.count = 1, .reusable = true}}, SHIFT(42),
  [35] = {.entry = {.count = 1, .reusable = true}}, SHIFT(41),
  [37] = {.entry = {.count = 1, .reusable = true}}, SHIFT(14),
  [39] = {.entry = {.count = 1, .reusable = true}}, SHIFT(16),
  [41] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_type_line, 6, 0, 7),
  [43] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_type_line, 6, 0, 7),
  [45] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_ref_line, 4, 0, 4),
  [47] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_ref_line, 4, 0, 4),
  [49] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment_line, 3, 0, 1),
  [51] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_comment_line, 3, 0, 1),
  [53] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment_line, 2, 0, 0),
  [55] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_comment_line, 2, 0, 0),
  [57] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_blob_line, 4, 0, 4),
  [59] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_blob_line, 4, 0, 4),
  [61] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_description_line, 4, 0, 2),
  [63] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_description_line, 4, 0, 2),
  [65] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_tag_line, 4, 0, 3),
  [67] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_tag_line, 4, 0, 3),
  [69] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_type_line, 4, 0, 5),
  [71] = {.entry = {.count = 1, .reusable = false}}, REDUCE(sym_type_line, 4, 0, 5),
  [73] = {.entry = {.count = 1, .reusable = true}}, SHIFT(36),
  [75] = {.entry = {.count = 1, .reusable = true}}, SHIFT(37),
  [77] = {.entry = {.count = 1, .reusable = false}}, SHIFT(40),
  [79] = {.entry = {.count = 1, .reusable = true}}, SHIFT(8),
  [81] = {.entry = {.count = 1, .reusable = true}}, SHIFT(26),
  [83] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_metadata, 4, 0, 0),
  [85] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_source_file, 1, 0, 0),
  [87] = {.entry = {.count = 1, .reusable = true}}, SHIFT(39),
  [89] = {.entry = {.count = 1, .reusable = true}}, SHIFT(12),
  [91] = {.entry = {.count = 1, .reusable = true}}, SHIFT(15),
  [93] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_metadata, 3, 0, 0),
  [95] = {.entry = {.count = 1, .reusable = true}}, SHIFT(30),
  [97] = {.entry = {.count = 1, .reusable = true}}, SHIFT(32),
  [99] = {.entry = {.count = 1, .reusable = true}}, SHIFT(34),
  [101] = {.entry = {.count = 1, .reusable = true}}, SHIFT(18),
  [103] = {.entry = {.count = 1, .reusable = true}}, SHIFT(20),
  [105] = {.entry = {.count = 1, .reusable = true}}, SHIFT(21),
  [107] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_comment_text, 1, 0, 0),
  [109] = {.entry = {.count = 1, .reusable = true}}, SHIFT(7),
  [111] = {.entry = {.count = 1, .reusable = true}},  ACCEPT_INPUT(),
  [113] = {.entry = {.count = 1, .reusable = true}}, SHIFT(3),
  [115] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_description_text, 1, 0, 0),
  [117] = {.entry = {.count = 1, .reusable = true}}, SHIFT(10),
  [119] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_tag_name, 1, 0, 0),
  [121] = {.entry = {.count = 1, .reusable = true}}, SHIFT(11),
  [123] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_reference_id, 1, 0, 0),
  [125] = {.entry = {.count = 1, .reusable = true}}, SHIFT(6),
  [127] = {.entry = {.count = 1, .reusable = true}}, SHIFT(9),
  [129] = {.entry = {.count = 1, .reusable = true}}, SHIFT(43),
  [131] = {.entry = {.count = 1, .reusable = true}}, SHIFT(22),
  [133] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_source_file, 2, 0, 0),
  [135] = {.entry = {.count = 1, .reusable = true}}, SHIFT(44),
  [137] = {.entry = {.count = 1, .reusable = true}}, SHIFT(13),
  [139] = {.entry = {.count = 1, .reusable = true}}, SHIFT(23),
  [141] = {.entry = {.count = 1, .reusable = true}}, SHIFT(46),
  [143] = {.entry = {.count = 1, .reusable = true}}, SHIFT(47),
  [145] = {.entry = {.count = 1, .reusable = true}}, SHIFT(5),
  [147] = {.entry = {.count = 1, .reusable = true}}, SHIFT(48),
  [149] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_markl_id, 3, 0, 6),
  [151] = {.entry = {.count = 1, .reusable = true}}, REDUCE(sym_markl_id, 4, 0, 8),
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

TS_PUBLIC const TSLanguage *tree_sitter_hyphence(void) {
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
