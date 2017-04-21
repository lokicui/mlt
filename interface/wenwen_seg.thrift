namespace cpp wenwen_seg
namespace php wenwen_seg

enum TermRelation
{
    ForceJoint,
    NeedJoint,
    SamePhrase,
    NextPhrase,
    EndTerm,
    MaxRelationShip,
}

struct QueryTermInfo
{
    1:i32   term_id,    // uint32 
    2:i32   term_wtype, // uint32
    3:i32   term_Ntype, // uint32
    4:i16   rel_flags,  // uint8
    5:i16   term_NImps, // uint8
    6:i16   term_RImps, // uint8
    7:i16   KCWeight,   // uint8
    8:i16   KCProb,     // uint8
    9:i16   pos,        // uint8
    10:i16  len,        // uint8
    11:i16  weight,     // uint8
    12:i16  importance, // uint8
    13:byte   relation,   // TermRelation
    14:i16  tightness,  // uint8
    15:byte  deletable, // char
}

struct TermRange
{
    1:i16   text_beg,  // index of query_bchart, query_gbk_sbc, uint8
    2:i16   text_end,
}

struct EntityInfo
{
    1:i16   term_beg,   // uint16
    2:i16   term_end,   // uint16
    3:i32   type,   // uint32
    4:i32   id,     // uint32
    5:i16   len,    // uint16
}

struct SynTermInfo
{
    1:i32   term_id,    
    2:byte  len,
    3:byte  flag,
}
struct SynTermWord
{
    1:i16   term_beg,  // uint8
    2:i16   term_end,  // uint8
    3:byte   len,   // char
    4:byte   type,  // char
    5:double    weight,
    6:double    context_weight,
}
struct SynTermMapping
{
    1:i16    org_beg,   // uint8
    2:i16    org_end,   // uint8
    3:byte   org_len,
    4:byte   org_flag,
    5:list<SynTermWord>    syn_words,
}

enum RetCode
{
	success = 0,
    error_busy = 1,
    error_query_too_long = 2,
    error_query_t2sgchar = 3,
	fail = 103,
}

struct QuerySegResponse
{
	1:RetCode retcode,
    2:list<TermRange>       annotation_terms,
    3:list<QueryTermInfo>   terms,
    4:list<EntityInfo>      entity_words;
    5:list<SynTermInfo>     syn_terms,
    6:list<SynTermMapping>  syn_term_mappings,
    7:list<list<byte> >      refined_query, // uint8, but only 0,1,2,3.
    8:list<list<byte> >      key_refined_query, // uint8, but only 0,1,2,3.
    9:string                 query_gbk_sbc,
    10:string                query_bchart,
}
struct ExtendTerm
{
    1:i32   termid,
    2:i16   level,  //uint8
    3:byte  type,   //char
    4:byte  pos,    //char
    5:double weight,    // float
}

service SegService
{
    // query_info 一次查询二次查询，推荐使用传入0即可
    QuerySegResponse    query_segment(1:string query_gbk, 2:i32 query_info),
    // from dicmap.
    i32     get_term_type(1:i32 term_id),
    i32     get_term_wtype(1:i32 term_id),
    i32     get_term_wf(1:i32   term_id),
    i32     get_term_qf(1:i32   term_id),
    i32     get_term_qef(1:i32  term_id),
    i32     get_term_delf(1:i32 term_id),
    string  get_term_text(1:i32   term_id),
    string  get_term_text_gbk(1:i32 term_id),
    string  postag2string(1:i32 wtype),
    list<ExtendTerm>    get_extend_term(1:i32 term_id),
}

