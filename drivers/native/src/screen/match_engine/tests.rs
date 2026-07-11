use super::super::types::{IconMatch, ScreenBoundingBox, ScreenPoint};
use super::{bbox_overlap_ratio, find_template_matches, item_position_order_bbox};
use std::cmp::Ordering;
use xcap::image::{Rgba, RgbaImage};

#[test]
fn find_template_matches_returns_ranked_deduped_candidates() {
    let mut haystack = RgbaImage::from_pixel(5, 5, Rgba([10, 10, 10, 255]));
    haystack.put_pixel(2, 1, Rgba([250, 250, 250, 255]));
    haystack.put_pixel(3, 1, Rgba([250, 250, 250, 255]));
    haystack.put_pixel(2, 2, Rgba([250, 250, 250, 255]));
    haystack.put_pixel(3, 2, Rgba([250, 250, 250, 255]));

    let template = RgbaImage::from_pixel(2, 2, Rgba([250, 250, 250, 255]));
    let matches = find_template_matches(&haystack, &template, 100, 200, 0.99, 3)
        .expect("match should succeed");
    assert_eq!(matches.len(), 1);
    assert_eq!(
        matches[0],
        IconMatch {
            score: 1.0,
            bbox: ScreenBoundingBox {
                x: 102,
                y: 201,
                width: 2,
                height: 2
            },
            center: ScreenPoint { x: 103, y: 202 },
            scale: 1.0
        }
    );
}

#[test]
fn bbox_overlap_ratio_uses_smaller_area_for_dedupe() {
    let overlap = bbox_overlap_ratio(
        &ScreenBoundingBox {
            x: 0,
            y: 0,
            width: 10,
            height: 10,
        },
        &ScreenBoundingBox {
            x: 2,
            y: 2,
            width: 6,
            height: 6,
        },
    );
    assert!(overlap > 0.9, "unexpected overlap ratio: {overlap}");
}

#[test]
fn item_position_order_bbox_prefers_top_then_left() {
    let left = ScreenBoundingBox {
        x: 10,
        y: 20,
        width: 1,
        height: 1,
    };
    let right = ScreenBoundingBox {
        x: 20,
        y: 20,
        width: 1,
        height: 1,
    };
    assert_eq!(item_position_order_bbox(&left, &right), Ordering::Less);
}
